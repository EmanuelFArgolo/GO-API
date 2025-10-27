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

// OllamaGenerateRequest is what Ollama /api/generate expects
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse is what Ollama returns (when stream: false)
type OllamaGenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"` // This will be a JSON string
	Done      bool      `json:"done"`
}

// --- Our Expected Output Structs ---

// LLMQuestionResponse is the format we *asked* the LLM to give us
type LLMQuestionResponse struct {
	Subject       string   `json:"subject"`
	QuestionText  string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
}

// LLMWrapper is to capture the {"questions": [...]} response from the LLM
type LLMWrapper struct {
	Questions []LLMQuestionResponse `json:"questions"`
}

// buildPrompt creates the actual text prompt for the LLM
func buildPrompt(theme string, wrongSubjects []string) string {
	subjectsStr := strings.Join(wrongSubjects, ", ")

	// Prompt pedindo 5 perguntas e o formato de objeto wrapper
	return fmt.Sprintf(`
	Crie um quiz de 10 perguntas sobre o tema principal '%s'.
	O foco principal do quiz deve ser nestes tópicos: %s.

	REGRAS DE FORMATAÇÃO DA RESPOSTA:
	1. Retorne APENAS um objeto JSON válido.
	2. O objeto JSON deve ter uma única chave "questions", que contém o array das perguntas. Exemplo: {"questions": [...]}.
	3. Não inclua NENHUM texto antes ou depois do objeto JSON (sem "Aqui está seu quiz:" ou markdown \`+"```"+`json).
	4. Cada objeto no array "questions" deve ter EXATAMENTE os seguintes campos:
	   - "subject": O tópico específico da pergunta.
	   - "question": O texto da pergunta.
	   - "options": Um array de 4 strings com as opções.
	   - "correct_answer": A string exata da opção correta.
	`, theme, subjectsStr)
}

// GenerateQuiz calls the Ollama endpoint
func (c *Client) GenerateQuiz(ctx context.Context, theme string, subjects []string) ([]LLMQuestionResponse, error) {

	// 1. Build the text prompt
	prompt := buildPrompt(theme, subjects)

	// 2. Create the Ollama-specific payload
	payload := OllamaGenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON for Ollama: %w", err)
	}

	// --- Logger "O que Sai" ---
	log.Println("-------------------------------------------------------")
	log.Printf("[LLM Request] Enviando para: %s", c.endpoint)
	log.Printf("[LLM Request] Payload: %s", string(payloadBytes))
	log.Println("-------------------------------------------------------")

	// 3. Send the request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for Ollama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// --- Logger "O que Volta" (Erro) ---
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("[LLM Response] ERRO: Status não-OK recebido: %s", res.Status)

		bodyBytes, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Printf("[LLM Response] Erro ao ler body da resposta de erro: %v", readErr)
		} else {
			log.Printf("[LLM Response] Body do Erro: %s", string(bodyBytes))
		}
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

		return nil, fmt.Errorf("Ollama returned non-OK status: %s", res.Status)
	}

	// 4. Decode the *Ollama* response
	var ollamaRes OllamaGenerateResponse
	if err := json.NewDecoder(res.Body).Decode(&ollamaRes); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	// 5. O JSON que queremos está DENTRO da string 'ollamaRes.Response'.
	log.Printf("[LLM Response] Raw string from LLM: %s", ollamaRes.Response)

	// Limpa a string (LLMs às vezes adicionam markdown)
	jsonString := strings.TrimSpace(ollamaRes.Response)
	if strings.HasPrefix(jsonString, "```json") {
		jsonString = strings.TrimPrefix(jsonString, "```json")
		jsonString = strings.TrimSuffix(jsonString, "```")
		jsonString = strings.TrimSpace(jsonString)
	}

	// Decodifica usando o Wrapper
	var wrappedResponse LLMWrapper
	if err := json.Unmarshal([]byte(jsonString), &wrappedResponse); err != nil {
		log.Printf("[LLM Error] O JSON da LLM estava quebrado/inválido. String: %s", jsonString)
		return nil, fmt.Errorf("failed to unmarshal wrapped JSON from LLM response: %w", err)
	}

	// Retorna o array de dentro do wrapper
	return wrappedResponse.Questions, nil
}
