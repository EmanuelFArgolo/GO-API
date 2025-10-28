// internal/service/service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"quizz-core/internal/llm"
	"quizz-core/internal/models"
	"quizz-core/internal/store"
	"strconv"
	"time"
)

var (
	ErrNotFound     = errors.New("recurso não encontrado")
	ErrInvalidInput = errors.New("input inválido")
)

// ... (struct QuizService e NewQuizService são iguais) ...
type QuizService struct {
	store     *store.Store
	llmClient *llm.Client
}

func NewQuizService(s *store.Store, llm *llm.Client) *QuizService {
	return &QuizService{
		store:     s,
		llmClient: llm,
	}
}

// CreateQuiz agora salva no banco de dados
func (s *QuizService) CreateQuiz(ctx context.Context, req models.CreateQuizRequest) (*models.QuizAPIResponse, error) {

	// 1. Chamar a LLM (igual a antes)
	llmQuestions, err := s.llmClient.GenerateQuiz(ctx, req.Theme, req.WrongSubjects)
	if err != nil {
		return nil, fmt.Errorf("service error calling LLM: %w", err)
	}

	// ===================================================================
	// === ADEUS, BLOCO MOCK! ===
	// ===================================================================
	// 2. Salvar o Quiz e as Perguntas no Banco
	quiz, perguntasSalvas, err := s.store.SaveGeneratedQuiz(ctx, req, llmQuestions)
	if err != nil {
		log.Printf("Erro ao salvar quiz no banco: %v", err)
		return nil, fmt.Errorf("falha ao salvar quiz no banco: %w", err)
	}

	// 3. Formatar a resposta da API (usando os dados reais do DB)
	var questionAPIs []models.QuestionAPI

	// Nota: O 'llmQuestions' tem as opções, 'perguntasSalvas' tem os IDs do DB
	for i, p := range perguntasSalvas {
		// Pega as opções da resposta da LLM
		llmOpt := llmQuestions[i].Options

		questionAPIs = append(questionAPIs, models.QuestionAPI{
			// Usamos o ID real da pergunta vindo do DB
			ID:       strconv.Itoa(p.ID), // Converte o 'int' do DB para 'string'
			Subject:  *p.Assunto,         // 'Assunto' é um ponteiro, usamos *
			Question: p.Corpo,
			Options:  llmOpt,
		})
	}

	// 4. Formatar e retornar a resposta final
	response := &models.QuizAPIResponse{
		// Usamos o ID real do quiz vindo do DB
		QuizID:    strconv.Itoa(quiz.ID), // Converte o 'int' do DB para 'string'
		Subject:   quiz.Nome,
		Questions: questionAPIs,
	}

	return response, nil
}

func (s *QuizService) SubmitAnswers(ctx context.Context, req models.SubmissionRequest) (*models.SubmissionResponse, error) {

	// 1. Converter IDs de String para Int
	// O nosso request da API usa strings, mas o DB usa integers.
	quizID, err := strconv.Atoi(req.QuizID)
	if err != nil {
		return nil, fmt.Errorf("quiz_id inválido: %w", ErrInvalidInput)
	}

	// (Vamos assumir que o user_id é um INT, apesar de vir como string)
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		// Numa app real, talvez o user_id seja um UUID/string.
		// Mas o nosso schema 'utilizadores' tem um ID SERIAL (int).
		return nil, fmt.Errorf("user_id inválido: %w", ErrInvalidInput)
	}

	// 2. Buscar o Gabarito (as respostas corretas) no DB
	// Chama a função que criámos no store
	answerMap, err := s.store.GetQuizAnswers(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar gabarito: %w", err)
	}
	if len(answerMap) == 0 {
		// Retorna o nosso erro personalizado
		return nil, fmt.Errorf("%w: quiz com id %d não encontrado ou não tem gabarito", ErrNotFound, quizID)
	}
	// 3. Inicializar contadores e listas para salvar
	correctCount := 0
	totalCount := len(answerMap)

	var respostasDadas []models.RespostaDada
	var dificuldades []models.Dificuldade

	// 4. Iterar sobre as Respostas do Utilizador e Comparar
	for _, userAnswer := range req.Answers {

		// Converte o QuestionID (string) da resposta do user para int
		questionID, err := strconv.Atoi(userAnswer.QuestionID)
		if err != nil {
			log.Printf("Aviso: question_id inválido recebido: %s", userAnswer.QuestionID)
			continue // Pula esta resposta
		}

		// Busca a resposta correta no nosso mapa
		correctAnswer, ok := answerMap[userAnswer.QuestionID]
		if !ok {
			log.Printf("Aviso: resposta recebida para uma pergunta que não pertence ao quiz: %s", userAnswer.QuestionID)
			continue // Pula esta resposta
		}

		// Compara a opção selecionada com o texto da opção correta
		isCorrect := (userAnswer.SelectedOption == correctAnswer.CorrectOption)

		// Atualiza contadores e listas
		if isCorrect {
			correctCount++
		} else {
			// Se errou, adiciona o 'assunto' à lista de dificuldades
			dificuldades = append(dificuldades, models.Dificuldade{
				Assunto: &correctAnswer.QuestionAssunto,
				// SubmissaoID será preenchido pelo store
			})
		}

		// Adiciona à lista de "respostas dadas"
		respostasDadas = append(respostasDadas, models.RespostaDada{
			PerguntaID:         questionID,
			CorretaNaSubmissao: &isCorrect,
			// SubmissaoID e RespostaID serão preenchidos pelo store
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
	// Chama a segunda função que criámos no store
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

func (s *QuizService) GetUserStats(ctx context.Context, userIDStr string) (*models.UserStatsResponse, error) {

	// 1. Validar e Converter Input
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		// Retorna o nosso erro personalizado para o handler apanhar
		return nil, fmt.Errorf("%w: user_id inválido", ErrInvalidInput)
	}

	// 2. Chamar o Store
	stats, err := s.store.GetUserStats(ctx, userID)
	if err != nil {
		// (O store.GetUserStats já trata o caso de 'ErrNoRows')
		return nil, fmt.Errorf("falha ao buscar estatísticas no serviço: %w", err)
	}

	// (Poderíamos verificar aqui se o utilizador existe, mas o GetUserStats já retorna zero, o que é bom)

	return stats, nil
}

// GetUserSubmissions é a lógica de negócios para buscar o histórico
func (s *QuizService) GetUserSubmissions(ctx context.Context, userIDStr string) ([]models.UserSubmissionHistoryResponse, error) {

	// 1. Validar e Converter Input
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: user_id inválido", ErrInvalidInput)
	}

	// 2. Chamar o Store
	submissions, err := s.store.GetUserSubmissions(ctx, userID)
	if err != nil {
		// (O store já trata o caso de 'ErrNoRows')
		return nil, fmt.Errorf("falha ao buscar histórico no serviço: %w", err)
	}

	return submissions, nil
}
