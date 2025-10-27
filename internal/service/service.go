package service

import (
	"context"
	"fmt"
	"quizz-core/internal/llm"
	"quizz-core/internal/models" // Uses the models package
	"quizz-core/internal/store"
)

// QuizService is our business logic struct
type QuizService struct {
	store     *store.Store // The database layer
	llmClient *llm.Client  // The LLM client layer
}

// NewQuizService is the constructor
func NewQuizService(s *store.Store, llm *llm.Client) *QuizService {
	return &QuizService{
		store:     s,
		llmClient: llm,
	}
}

// CreateQuiz orchestrates the creation of a quiz
func (s *QuizService) CreateQuiz(ctx context.Context, req models.CreateQuizRequest) (*models.QuizAPIResponse, error) {

	// 1. Call the LLM to generate questions
	llmQuestions, err := s.llmClient.GenerateQuiz(ctx, req.Theme, req.WrongSubjects)
	if err != nil {
		return nil, fmt.Errorf("service error calling LLM: %w", err)
	}

	// 2. Save the Quiz and Questions to the Database
	// TODO: Implement the saving logic in the 'store'.
	// The 'store' will need a transaction to:
	//    a. Create a 'Tema' if it doesn't exist (or fetch it)
	//    b. Create a 'Quiz' linked to the 'Tema'
	//    c. Create the 'Perguntas' linked to the 'Quiz'
	//    d. Create the 'Respostas' linked to each 'Pergunta'

	// quizID, questionAPIs, err := s.store.SaveGeneratedQuiz(ctx, req, llmQuestions)
	// if err != nil {
	// 	return nil, err
	// }

	// 3. (Mocked for now)
	// ----- START MOCK (replace this with DB logic) -----
	quizID := "mock-quiz-12345" // Fake ID
	var questionAPIs []models.QuestionAPI
	for i, q := range llmQuestions {
		questionAPIs = append(questionAPIs, models.QuestionAPI{
			ID:       fmt.Sprintf("q%d", i+1),
			Subject:  q.Subject,
			Question: q.QuestionText,
			Options:  q.Options,
		})
	}
	// ----- END MOCK -----

	// 4. Format and return the API response
	response := &models.QuizAPIResponse{
		QuizID:    quizID,
		Subject:   req.Theme,
		Questions: questionAPIs,
	}

	return response, nil
}
