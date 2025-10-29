package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"quizz-core/internal/models"
	"quizz-core/internal/service" // Import o service
)

// SimpleMessageResponse é uma struct genérica para respostas de sucesso
type SimpleMessageResponse struct {
	Message string `json:"message"`
}

// ApiHandlers is our struct that will hold dependencies
type ApiHandlers struct {
	quizService *service.QuizService
}

// NewApiHandlers is the constructor for our handlers
func NewApiHandlers(qs *service.QuizService) *ApiHandlers {
	return &ApiHandlers{
		quizService: qs,
	}
}

// HealthCheckHandler (agora verifica as dependências)
func (h *ApiHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	healthStatus := h.quizService.CheckHealth(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if healthStatus.Dependencies["database"] == models.StatusDown {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
	} else {
		w.WriteHeader(http.StatusOK) // 200
	}
	if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
		log.Printf("Erro ao enviar resposta JSON de health check: %v", err)
		http.Error(w, "Erro ao gerar health status", http.StatusInternalServerError)
	}
}

// CreateQuizHandler agora retorna o JSON cru da LLM
func (h *ApiHandlers) CreateQuizHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode JSON
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

	// 4. Call the Service
	// Agora recebe RawQuizResponse em vez de QuizAPIResponse
	rawQuizResponse, err := h.quizService.CreateQuiz(r.Context(), req)
	if err != nil {
		log.Printf("Error creating quiz raw: %v", err)
		http.Error(w, "Falha ao gerar quiz", http.StatusInternalServerError)
		return
	}

	// 5. Send the raw JSON response back
	w.Header().Set("Content-Type", "application/json")
	// Usamos 200 OK porque não criámos um recurso persistente *neste serviço*
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(rawQuizResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON crua: %v", err)
	}
}

// --- OS HANDLERS ABAIXO NÃO MUDAM ---
// (SubmitAnswersHandler, GetUserStatsHandler, GetUserSubmissionsHandler, GetAllThemesHandler, DeactivateQuizHandler, GetSubmissionDetailsHandler, GetQuizzesByThemeHandler)

// SubmitAnswersHandler é o handler para receber as respostas do quiz
func (h *ApiHandlers) SubmitAnswersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req models.SubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

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

	subResponse, err := h.quizService.SubmitAnswers(r.Context(), req)

	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("Erro 404 em SubmitAnswers: %v", err)
			http.Error(w, "Recurso não encontrado (ex: quiz_id ou user_id inválido)", http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em SubmitAnswers: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em SubmitAnswers: %v", err)
			http.Error(w, "Falha interna ao processar submissão", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(subResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON da submissão: %v", err)
	}
}

// GetUserStatsHandler é o handler para as estatísticas do utilizador
func (h *ApiHandlers) GetUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Input inválido: 'user_id' é obrigatório (query parameter)", http.StatusBadRequest)
		return
	}

	statsResponse, err := h.quizService.GetUserStats(r.Context(), userID)

	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em GetUserStats: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em GetUserStats: %v", err)
			http.Error(w, "Falha interna ao buscar estatísticas", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statsResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON de estatísticas: %v", err)
	}
}

// GetUserSubmissionsHandler é o handler para o histórico de submissões
func (h *ApiHandlers) GetUserSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Input inválido: 'user_id' é obrigatório (query parameter)", http.StatusBadRequest)
		return
	}

	submissionsResponse, err := h.quizService.GetUserSubmissions(r.Context(), userID)

	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em GetUserSubmissions: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em GetUserSubmissions: %v", err)
			http.Error(w, "Falha interna ao buscar histórico", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(submissionsResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON de histórico: %v", err)
	}
}

// GetAllThemesHandler é o handler para listar todos os temas ativos
func (h *ApiHandlers) GetAllThemesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	temas, err := h.quizService.GetAllActiveThemes(r.Context())
	if err != nil {
		log.Printf("Erro 500 em GetAllThemes: %v", err)
		http.Error(w, "Falha interna ao buscar temas", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(temas); err != nil {
		log.Printf("Erro ao enviar resposta JSON de temas: %v", err)
	}
}

// DeactivateQuizHandler é o handler para o "soft-delete" de um quiz
func (h *ApiHandlers) DeactivateQuizHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método não permitido, use PUT", http.StatusMethodNotAllowed)
		return
	}
	quizID := r.URL.Query().Get("quiz_id")
	if quizID == "" {
		http.Error(w, "Input inválido: 'quiz_id' é obrigatório (query parameter)", http.StatusBadRequest)
		return
	}

	err := h.quizService.DeactivateQuiz(r.Context(), quizID)

	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("Erro 404 em DeactivateQuiz: %v", err)
			http.Error(w, "Recurso não encontrado (quiz_id não existe ou já está inativo)", http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em DeactivateQuiz: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em DeactivateQuiz: %v", err)
			http.Error(w, "Falha interna ao desativar quiz", http.StatusInternalServerError)
		}
		return
	}

	response := SimpleMessageResponse{Message: "Quiz desativado com sucesso"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Erro ao enviar resposta JSON de desativação: %v", err)
	}
}

// GetSubmissionDetailsHandler é o handler para os detalhes de uma submissão
func (h *ApiHandlers) GetSubmissionDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	submissionID := r.URL.Query().Get("submission_id")
	if submissionID == "" {
		http.Error(w, "Input inválido: 'submission_id' é obrigatório (query parameter)", http.StatusBadRequest)
		return
	}

	detailsResponse, err := h.quizService.GetSubmissionDetails(r.Context(), submissionID)

	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("Erro 404 em GetSubmissionDetails: %v", err)
			http.Error(w, "Recurso não encontrado (submission_id não existe)", http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em GetSubmissionDetails: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em GetSubmissionDetails: %v", err)
			http.Error(w, "Falha interna ao buscar detalhes da submissão", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(detailsResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON de detalhes da submissão: %v", err)
	}
}

// GetQuizzesByThemeHandler é o handler para listar quizzes de um tema
func (h *ApiHandlers) GetQuizzesByThemeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	themeID := r.URL.Query().Get("theme_id")
	if themeID == "" {
		http.Error(w, "Input inválido: 'theme_id' é obrigatório (query parameter)", http.StatusBadRequest)
		return
	}

	quizzesResponse, err := h.quizService.GetActiveQuizzesByTheme(r.Context(), themeID)

	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			log.Printf("Erro 400 em GetQuizzesByTheme: %v", err)
			http.Error(w, "Input inválido: "+err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Erro 500 em GetQuizzesByTheme: %v", err)
			http.Error(w, "Falha interna ao buscar quizzes", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(quizzesResponse); err != nil {
		log.Printf("Erro ao enviar resposta JSON de quizzes: %v", err)
	}
}
