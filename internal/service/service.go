package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"quizz-core/internal/llm"
	"quizz-core/internal/models"
	"quizz-core/internal/store"
	"strconv"
	"time"
)

// --- Erros Personalizados de Negócios ---
var (
	ErrNotFound     = errors.New("recurso não encontrado")
	ErrInvalidInput = errors.New("input inválido")
)

// ------------------------------------

// QuizService is our business logic struct
type QuizService struct {
	store     *store.Store // The database layer (ainda necessário para Submit e GETs)
	llmClient *llm.Client  // The LLM client layer
}

// NewQuizService is the constructor
func NewQuizService(s *store.Store, llm *llm.Client) *QuizService {
	return &QuizService{
		store:     s,
		llmClient: llm,
	}
}

// CreateQuiz agora apenas chama a LLM e retorna o JSON cru
func (s *QuizService) CreateQuiz(ctx context.Context, req models.CreateQuizRequest) (*models.RawQuizResponse, error) { // <-- Retorna RawQuizResponse

	// 1. Chamar a LLM para obter a string JSON
	rawJsonString, err := s.llmClient.GenerateQuiz(ctx, req.Theme, req.WrongSubjects)
	if err != nil {
		// Mantém o log de erro interno
		log.Printf("Erro ao gerar quiz via LLM: %v", err)
		// Retorna um erro genérico para o handler (que dará 500)
		return nil, fmt.Errorf("service error calling LLM: %w", err)
	}

	// === NÃO HÁ MAIS CHAMADA AO STORE AQUI PARA SALVAR O QUIZ ===
	// quiz, perguntasSalvas, err := s.store.SaveGeneratedQuiz(...)
	// ==========================================================

	// 2. Formatar a nova resposta (RawQuizResponse)
	response := &models.RawQuizResponse{
		UserID:     req.UserID,    // Passa o UserID original
		RawLLMJson: rawJsonString, // Passa a string JSON crua
	}

	return response, nil
}

// --- AS FUNÇÕES ABAIXO NÃO MUDAM ---
// (SubmitAnswers, GetUserStats, GetUserSubmissions, GetAllActiveThemes, DeactivateQuiz, GetSubmissionDetails, CheckHealth)

// SubmitAnswers é a lógica de negócios para processar as respostas de um quiz
func (s *QuizService) SubmitAnswers(ctx context.Context, req models.SubmissionRequest) (*models.SubmissionResponse, error) {

	// 1. Converter IDs de String para Int
	quizID, err := strconv.Atoi(req.QuizID)
	if err != nil {
		return nil, fmt.Errorf("%w: quiz_id inválido", ErrInvalidInput)
	}
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: user_id inválido", ErrInvalidInput)
	}

	// 2. Buscar o Gabarito (agora retorna o mapa complexo)
	answerMap, err := s.store.GetQuizAnswers(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar gabarito: %w", err)
	}
	if len(answerMap) == 0 {
		return nil, fmt.Errorf("%w: quiz com id %d não encontrado", ErrNotFound, quizID)
	}

	// 3. Inicializar contadores e listas para salvar
	correctCount := 0
	totalCount := len(answerMap)
	var respostasDadas []models.RespostaDada
	var dificuldades []models.Dificuldade

	// 4. Iterar sobre as Respostas do Utilizador e Comparar (LÓGICA ATUALIZADA)
	for _, userAnswer := range req.Answers {
		questionID, err := strconv.Atoi(userAnswer.QuestionID)
		if err != nil {
			log.Printf("Aviso: question_id inválido recebido: %s", userAnswer.QuestionID)
			continue
		}

		questionInfo, ok := answerMap[userAnswer.QuestionID]
		if !ok {
			log.Printf("Aviso: resposta recebida para uma pergunta que não pertence ao quiz: %s", userAnswer.QuestionID)
			continue
		}

		// Compara o texto
		isCorrect := (userAnswer.SelectedOption == questionInfo.CorrectOptionText)

		var selectedAnswerID *int // Começa como nulo
		if id, ok := questionInfo.OptionsMap[userAnswer.SelectedOption]; ok {
			selectedAnswerID = &id // Encontrámos o ID!
		} else {
			log.Printf("Aviso: Opção '%s' não encontrada para a pergunta %d", userAnswer.SelectedOption, questionID)
		}

		if isCorrect {
			correctCount++
		} else {
			dificuldades = append(dificuldades, models.Dificuldade{
				Assunto: &questionInfo.Assunto,
			})
		}

		// Adiciona à lista de "respostas dadas" com o ID correto
		respostasDadas = append(respostasDadas, models.RespostaDada{
			PerguntaID:         questionID,
			CorretaNaSubmissao: &isCorrect,
			RespostaID:         selectedAnswerID, // <-- GUARDAMOS O ID
		})
	}

	// 5. Calcular Pontuação
	var score float64 = 0
	if totalCount > 0 {
		score = (float64(correctCount) / float64(totalCount)) * 100.0
	}

	// 6. Preparar o objeto 'Submissao' para salvar
	submissaoParaSalvar := models.Submissao{
		DataHora:     time.Now(),
		Pontuacao:    score,
		UtilizadorID: userID,
		QuizzID:      quizID,
	}

	// 7. Salvar Tudo no DB (em uma única transação)
	savedSub, err := s.store.SaveSubmissionStats(ctx, submissaoParaSalvar, respostasDadas, dificuldades)
	if err != nil {
		return nil, fmt.Errorf("falha ao salvar estatísticas da submissão: %w", err)
	}

	// 8. Formatar a Resposta da API
	response := &models.SubmissionResponse{
		SubmissionID: savedSub.ID,
		Score:        savedSub.Pontuacao,
		CorrectCount: correctCount,
		TotalCount:   totalCount,
		Message:      fmt.Sprintf("Submissão bem-sucedida! Acertou %d de %d.", correctCount, totalCount),
	}

	return response, nil
}

// GetUserStats é a lógica de negócios para buscar as estatísticas
func (s *QuizService) GetUserStats(ctx context.Context, userIDStr string) (*models.UserStatsResponse, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: user_id inválido", ErrInvalidInput)
	}
	stats, err := s.store.GetUserStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar estatísticas no serviço: %w", err)
	}
	return stats, nil
}

// GetUserSubmissions é a lógica de negócios para buscar o histórico
func (s *QuizService) GetUserSubmissions(ctx context.Context, userIDStr string) ([]models.UserSubmissionHistoryResponse, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: user_id inválido", ErrInvalidInput)
	}
	submissions, err := s.store.GetUserSubmissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar histórico no serviço: %w", err)
	}
	return submissions, nil
}

// GetAllActiveThemes é a lógica de negócios para buscar os temas
func (s *QuizService) GetAllActiveThemes(ctx context.Context) ([]models.Tema, error) {
	temas, err := s.store.GetAllActiveThemes(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar temas ativos no serviço: %w", err)
	}
	return temas, nil
}

// DeactivateQuiz é a lógica de negócios para o soft-delete
func (s *QuizService) DeactivateQuiz(ctx context.Context, quizIDStr string) error {
	quizID, err := strconv.Atoi(quizIDStr)
	if err != nil {
		return fmt.Errorf("%w: quiz_id inválido", ErrInvalidInput)
	}
	rowsAffected, err := s.store.DeactivateQuiz(ctx, quizID)
	if err != nil {
		return fmt.Errorf("falha ao desativar quiz no serviço: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: quiz com id %d não encontrado ou já está inativo", ErrNotFound, quizID)
	}
	return nil
}

// GetSubmissionDetails é a lógica de negócios para buscar os detalhes
func (s *QuizService) GetSubmissionDetails(ctx context.Context, submissionIDStr string) (*models.SubmissionDetailResponse, error) {
	submissionID, err := strconv.Atoi(submissionIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: submission_id inválido", ErrInvalidInput)
	}
	details, err := s.store.GetSubmissionDetails(ctx, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: submissão com id %d não encontrada", ErrNotFound, submissionID)
		}
		return nil, fmt.Errorf("falha ao buscar detalhes da submissão no serviço: %w", err)
	}
	return details, nil
}

// GetActiveQuizzesByTheme é a lógica de negócios para buscar quizzes
func (s *QuizService) GetActiveQuizzesByTheme(ctx context.Context, themeIDStr string) ([]models.Quiz, error) {
	themeID, err := strconv.Atoi(themeIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: theme_id inválido", ErrInvalidInput)
	}
	quizzes, err := s.store.GetActiveQuizzesByTheme(ctx, themeID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar quizzes por tema no serviço: %w", err)
	}
	return quizzes, nil
}

// CheckHealth verifica o estado das dependências (DB e LLM)
func (s *QuizService) CheckHealth(ctx context.Context) models.HealthResponse {
	response := models.HealthResponse{
		OverallStatus: models.StatusUp,
		Dependencies:  make(map[string]models.HealthStatus),
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.store.Ping(checkCtx); err != nil {
		log.Printf("Health Check ERRO: Falha ao pingar DB: %v", err)
		response.Dependencies["database"] = models.StatusDown
		response.OverallStatus = models.StatusDown
	} else {
		response.Dependencies["database"] = models.StatusUp
	}

	if err := s.llmClient.Ping(checkCtx); err != nil {
		log.Printf("Health Check AVISO: Falha ao pingar LLM: %v", err)
		response.Dependencies["llm"] = models.StatusDown
	} else {
		response.Dependencies["llm"] = models.StatusUp
	}
	return response
}
