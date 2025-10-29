package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client is our struct for the LLM client
type Client struct {
	httpClient *http.Client
	endpoint   string
	model      string
}

// NewClient is the constructor for our LLM client
func NewClient(endpoint, model string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // 3-minute timeout
		},
		endpoint: endpoint,
		model:    model,
	}
}

// --- Ollama Specific Structs ---
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type OllamaGenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"` // This will be a JSON string
	Done      bool      `json:"done"`
}

// --- Our Expected Output Structs (NÃO USADAS NESTE CENÁRIO POR GenerateQuiz) ---
type LLMQuestionResponse struct {
	Subject       string   `json:"subject"`
	QuestionText  string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	Explanation   string   `json:"explanation"`
}
type LLMWrapper struct {
	Title     string                `json:"title"`
	Questions []LLMQuestionResponse `json:"questions"`
}

// buildPrompt (ajuste o número de perguntas conforme necessário)
func buildPrompt(theme string, wrongSubjects []string) string {
	subjectsStr := strings.Join(wrongSubjects, ", ")
	return fmt.Sprintf(`
	Crie um quiz de 1 perguntas sobre o tema principal '%s'.
	O foco principal do quiz deve ser nestes tópicos: %s.

	REGRAS DE FORMATAÇÃO DA RESPOSTA:
	1. Retorne APENAS um objeto JSON válido.
	2. O objeto JSON deve ter DUAS chaves: "title" (string) e "questions" (array). Ex: {"title": "Título", "questions": [...]}.
	3. Não inclua NENHUM texto antes ou depois do objeto JSON (sem markdown \`+"```"+`json).
	4. Cada objeto no array "questions" deve ter EXATAMENTE os seguintes campos:
	   - "subject": O tópico específico da pergunta.
	   - "question": O texto da pergunta.
	   - "options": Um array de 4 strings com as opções.
	   - "correct_answer": A string exata da opção correta.
	   - "explanation": Uma string curta explicando PORQUÊ a resposta correta está certa. <-- NOVA REGRA
	5. Gere o JSON completo e válido.
	`, theme, subjectsStr)
}

// Ping verifica a conectividade com o servidor LLM (Ollama)
func (c *Client) Ping(ctx context.Context) error {
	pingClient := http.Client{
		Timeout: 5 * time.Second, // Timeout curto de 5 segundos
	}
	baseURL := strings.Split(c.endpoint, "/api/")[0]
	if baseURL == "" {
		baseURL = c.endpoint // Fallback
	}
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return fmt.Errorf("falha ao criar request de ping para LLM: %w", err)
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao enviar ping para LLM: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// GenerateQuiz agora retorna a string JSON limpa
func (c *Client) GenerateQuiz(ctx context.Context, theme string, subjects []string) (string, error) { // <-- Returns string, error

	// 1. Build the text prompt (which now asks for a title)
	prompt := buildPrompt(theme, subjects)

	// 2. Create the Ollama-specific payload
	payload := OllamaGenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON for Ollama: %w", err)
	}

	// --- Logger "What Goes Out" ---
	log.Println("-------------------------------------------------------")
	log.Printf("[LLM Request] Enviando para: %s", c.endpoint)
	log.Printf("[LLM Request] Payload: %s", string(payloadBytes))
	log.Println("-------------------------------------------------------")

	// 3. Send the request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request for Ollama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// --- Logger "What Comes Back" (Error) ---
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("[LLM Response] ERRO: Status não-OK recebido: %s", res.Status)
		bodyBytes, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Printf("[LLM Response] Erro ao ler body da resposta de erro: %v", readErr)
		} else {
			log.Printf("[LLM Response] Body do Erro: %s", string(bodyBytes))
		}
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		return "", fmt.Errorf("Ollama returned non-OK status: %s", res.Status) // Return "" on error
	}

	// 4. Decode the *Ollama* response
	var ollamaRes OllamaGenerateResponse
	if err := json.NewDecoder(res.Body).Decode(&ollamaRes); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err) // Return "" on error
	}

	// 5. The JSON we want is INSIDE the 'ollamaRes.Response' string.
	log.Printf("[LLM Response] Raw string from LLM: %s", ollamaRes.Response)

	// --- Bulletproof Cleaning Logic ---
	jsonString := ollamaRes.Response
	firstBracket := strings.Index(jsonString, "{")
	if firstBracket == -1 {
		log.Printf("[LLM Error] JSON da LLM não continha um '{'.")
		return "", fmt.Errorf("LLM response did not contain JSON opening bracket") // Return "" on error
	}
	lastBracket := strings.LastIndex(jsonString, "}")
	if lastBracket == -1 {
		log.Printf("[LLM Error] JSON da LLM não continha um '}'.")
		return "", fmt.Errorf("LLM response did not contain JSON closing bracket") // Return "" on error
	}
	jsonString = jsonString[firstBracket : lastBracket+1]
	log.Printf("[LLM Response] Cleaned JSON string to attempt parsing: %s", jsonString) // Log before trying to parse
	// --- End Cleaning Logic ---

	// === Attempt to Parse to Verify Structure (including Title) ===
	// We parse it here to LOG if it fails, but we still return the string
	var wrappedResponse LLMWrapper // Uses the struct with Title and Questions
	if err := json.Unmarshal([]byte(jsonString), &wrappedResponse); err != nil {
		log.Printf("[LLM Error] O JSON da LLM estava quebrado/inválido (apesar da limpeza). String: %s. Erro: %v", jsonString, err)
		// Even if parsing fails here, we return the cleaned string for API1 to try.
		return jsonString, nil // Return the cleaned string anyway
	}
	// If parsing worked, log success and potentially re-encode for safety (optional but safer)
	log.Printf("[LLM Info] JSON parse successful, found title: '%s'", wrappedResponse.Title)
	finalJsonBytes, err := json.Marshal(wrappedResponse)
	if err != nil {
		log.Printf("Erro ao re-codificar JSON verificado: %v", err)
		return jsonString, nil // Fallback to original cleaned string
	}
	// Return the verified and re-encoded JSON string
	return string(finalJsonBytes), nil
	// =================================================================

}
