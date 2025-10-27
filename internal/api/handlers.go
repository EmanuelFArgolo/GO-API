package api

import (
	"encoding/json"
	"errors" // <-- Import necessário
	"fmt"
	"log"
	"net/http"
	"quizz-core/internal/models"
	"quizz-core/internal/service" // Import o service
)

// ApiHandlers é a nossa struct que vai segurar as dependências,
// like the quiz service.
type ApiHandlers struct {
	quizService *service.QuizService
}

// NewApiHandlers é o construtor para nossos handlers
func NewApiHandlers(qs *service.QuizService) *ApiHandlers {
	// It receives the service (dependency injection)
	return &ApiHandlers{
		quizService: qs,
	}
}

// HealthCheckHandler (now a method of ApiHandlers)
func (h *ApiHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok"}`)
}

// CreateQuizHandler é o handler para o seu endpoint principal
// It knows about HTTP, the service does not.
func (h *ApiHandlers) CreateQuizHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only accept POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the JSON from the request body into our struct
	var req models.CreateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Validação de Input
	if req.UserID == "" {
		http.Error(w, "Input inválido: 'user_id' não pode estar em branco", http.StatusBadRequest)
		return
	}
	if req.Theme == "" {
		http.Error(w, "Input inválido: 'theme' não pode estar em branco", http.StatusBadRequest)
		return
	}

	// 4. Call the Service (the "brain")
	quizResponse, err := h.quizService.CreateQuiz(r.Context(), req)
	if err != nil {
		// (Aqui podemos também verificar erros personalizados, mas por enquanto um 500 basta)
		log.Printf("Error creating quiz: %v", err)
		http.Error(w, "Falha ao criar quiz", http.StatusInternalServerError)
		return
	}

	// 5. Send the successful JSON response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created is a good status here
	if err := json.NewEncoder(w).Encode(quizResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON: %v", err)
	}
}

// SubmitAnswersHandler é o handler para receber as respostas do quiz
func (h *ApiHandlers) SubmitAnswersHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only accept POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the JSON
	var req models.SubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Validação de Input
	if req.QuizID == "" {
		http.Error(w, "Input inválido: 'quiz_id' não pode estar em branco", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "Input inválido: 'user_id' não pode estar em branco", http.StatusBadRequest)
		return
	}
	if len(req.Answers) == 0 {
		http.Error(w, "Input inválido: 'answers' não pode estar vazio", http.StatusBadRequest)
		return
	}

	// 4. Call the Service (the "brain")
	subResponse, err := h.quizService.SubmitAnswers(r.Context(), req)

	// 5. Tratamento Avançado de Erros
	if err != nil {
		// Verificamos o *tipo* de erro que o serviço nos deu
		if errors.Is(err, service.ErrNotFound) {
			// Erro 404: O utilizador pediu um quiz que não existe
			log.Printf("Erro 404 em SubmitAnswers: %v", err)
			http.Error(w, "Recurso não encontrado (ex: quiz_id ou user_id inválido)", http.StatusNotFound)

		} else if errors.Is(err, service.ErrInvalidInput) {
			// Erro 400: O input era semanticamente inválido (ex: user_id="abc")
			log.Printf("Erro 400 em SubmitAnswers: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)

		} else {
			// Erro 500: Um erro do nosso lado (DB offline, etc.)
			log.Printf("Erro 500 em SubmitAnswers: %v", err)
			http.Error(w, "Falha interna ao processar submissão", http.StatusInternalServerError)
		}
		return
	}

	// 6. Envia a resposta (os resultados)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK
	if err := json.NewEncoder(w).Encode(subResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON da submissão: %v", err)
	}
}
