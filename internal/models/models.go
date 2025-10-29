package models

import (
	"time"
)

// --- Database Structs (Based on your SQL) ---

// Utilizador (from 'utilizadores' table)
type Utilizador struct {
	ID             int        `json:"id" db:"id"`
	Nome           string     `json:"nome" db:"nome"`
	Password       string     `json:"-" db:"password"` // Hidden in JSON
	Tipo           string     `json:"tipo" db:"tipo"`
	DataNascimento *time.Time `json:"data_nascimento,omitempty" db:"datanascimento"`
	Criacao        time.Time  `json:"criacao" db:"criacao"`
}

// Tema (from 'tema' table)
type Tema struct {
	ID        int       `json:"id" db:"id"`
	Nome      string    `json:"nome" db:"nome"`
	Descricao *string   `json:"descricao,omitempty" db:"descricao"`
	Criacao   time.Time `json:"criacao" db:"criacao"`
	Ativo     bool      `json:"ativo" db:"ativo"`
}

// Quiz (from 'quizzes' table)
type Quiz struct {
	ID     int    `json:"id" db:"id"`
	Nome   string `json:"nome" db:"nome"`
	TemaID int    `json:"tema_id" db:"tema_id"`
	Ativo  bool   `json:"ativo" db:"ativo"`
}

// Pergunta (from 'perguntas' table)
type Pergunta struct {
	ID         int     `json:"id" db:"id"`
	Assunto    *string `json:"assunto,omitempty" db:"assunto"`
	Corpo      string  `json:"corpo" db:"corpo"`
	Explicacao *string `json:"explicacao,omitempty" db:"explicacao"`
	QuizzID    int     `json:"quizz_id" db:"quizz_id"`
}

// Resposta (from 'respostas' table)
type Resposta struct {
	ID         int    `json:"id" db:"id"`
	Corpo      string `json:"corpo" db:"corpo"`
	Correta    bool   `json:"correta" db:"correta"`
	PerguntaID int    `json:"pergunta_id" db:"pergunta_id"`
}

// Submissao (from 'submissao' table)
type Submissao struct {
	ID           int       `json:"id" db:"id"`
	DataHora     time.Time `json:"data_hora" db:"datahora"`
	Pontuacao    float64   `json:"pontuacao" db:"pontuacao"`
	UtilizadorID int       `json:"utilizador_id" db:"utilizador_id"`
	QuizzID      int       `json:"quizz_id" db:"quizz_id"`
}

// Dificuldade (from 'dificuldades' table)
type Dificuldade struct {
	ID          int     `json:"id" db:"id"`
	Assunto     *string `json:"assunto,omitempty" db:"assunto"`
	SubmissaoID int     `json:"submissao_id" db:"submissao_id"`
}

// RespostaDada (from 'respostas_dadas' table)
type RespostaDada struct {
	ID                 int   `json:"id" db:"id"`
	SubmissaoID        int   `json:"submissao_id" db:"submissao_id"`
	PerguntaID         int   `json:"pergunta_id" db:"pergunta_id"`
	RespostaID         *int  `json:"resposta_id,omitempty" db:"resposta_id"`
	CorretaNaSubmissao *bool `json:"correta_na_submissao,omitempty" db:"correta_na_submissao"`
}

// --- API Structs (What comes in and what goes out) ---

// CreateQuizRequest is what your API will receive from the other API
type CreateQuizRequest struct {
	UserID        string   `json:"user_id"`        // Using string as requested
	Theme         string   `json:"theme"`          // e.g., "Physics"
	WrongSubjects []string `json:"wrong_subjects"` // The 5 subjects
}

// QuizAPIResponse is what you will return
type QuizAPIResponse struct {
	QuizID    string        `json:"quiz_id"` // The ID of the generated quiz
	Subject   string        `json:"subject"` // The general theme
	Questions []QuestionAPI `json:"questions"`
}

// QuestionAPI is the question struct formatted for the JSON response
type QuestionAPI struct {
	ID       string   `json:"id"`       // "q1", "q2"
	Subject  string   `json:"subject"`  // The specific subject
	Question string   `json:"question"` // The question text
	Options  []string `json:"options"`  // The options
}

type SubmissionRequest struct {
	QuizID  string       `json:"quiz_id"`
	UserID  string       `json:"user_id"` // Usando string para consistência
	Answers []UserAnswer `json:"answers"`
}

// UserAnswer é a resposta de uma única pergunta
type UserAnswer struct {
	QuestionID     string `json:"question_id"`     // O ID da pergunta (ex: "1", "2")
	SelectedOption string `json:"selected_option"` // O *texto* da opção que o usuário escolheu
}

// SubmissionResponse é o que retornamos após a submissão
type SubmissionResponse struct {
	SubmissionID int     `json:"submission_id"`
	Score        float64 `json:"score"`
	CorrectCount int     `json:"correct_count"`
	TotalCount   int     `json:"total_count"`
	Message      string  `json:"message"`
}
type UserStatsResponse struct {
	UserID                    string  `json:"user_id"`
	TotalQuizzesRealizados    int     `json:"total_quizzes_realizados"`
	TotalPerguntasRespondidas int     `json:"total_perguntas_respondidas"`
	TotalAcertos              int     `json:"total_acertos"`
	TotalErros                int     `json:"total_erros"`
	PercentagemAcerto         float64 `json:"percentagem_acerto"`
	PontuacaoMedia            float64 `json:"pontuacao_media"`
}
type UserSubmissionHistoryResponse struct {
	SubmissionID int       `json:"submission_id"`
	QuizID       int       `json:"quiz_id"`
	QuizNome     string    `json:"quiz_nome"` // O nome do quiz
	TemaNome     string    `json:"tema_nome"` // O nome do tema
	Pontuacao    float64   `json:"pontuacao"` // A pontuação obtida
	DataHora     time.Time `json:"data_hora"` // Quando foi feito
}

// SubmissionDetailResponse é a resposta completa para o endpoint de detalhes
type SubmissionDetailResponse struct {
	SubmissionID int                      `json:"submission_id"`
	QuizID       int                      `json:"quiz_id"`
	QuizNome     string                   `json:"quiz_nome"`
	TemaNome     string                   `json:"tema_nome"`
	Pontuacao    float64                  `json:"pontuacao"`
	DataHora     time.Time                `json:"data_hora"`
	Perguntas    []QuestionDetailResponse `json:"perguntas"` // Array com os detalhes de cada pergunta
}

// QuestionDetailResponse contém os detalhes de uma pergunta dentro da submissão
type QuestionDetailResponse struct {
	PerguntaID         int                  `json:"pergunta_id"`
	CorpoPergunta      string               `json:"corpo_pergunta"`
	Assunto            *string              `json:"assunto,omitempty"`
	Opcoes             []AnswerOptionDetail `json:"opcoes"`                        // Todas as opções da pergunta
	RespostaUtilizador *string              `json:"resposta_utilizador,omitempty"` // O texto da opção que o user escolheu
	RespostaCorreta    string               `json:"resposta_correta"`              // O texto da opção correta
	Acertou            bool                 `json:"acertou"`                       // Se o user acertou esta pergunta
}

// AnswerOptionDetail representa uma única opção de resposta
type AnswerOptionDetail struct {
	RespostaID int    `json:"resposta_id"`
	Corpo      string `json:"corpo"`
}

type HealthStatus string

const (
	StatusUp   HealthStatus = "UP"
	StatusDown HealthStatus = "DOWN"
)

// HealthResponse é a resposta detalhada do endpoint /health
type HealthResponse struct {
	OverallStatus HealthStatus            `json:"status"`
	Dependencies  map[string]HealthStatus `json:"dependencies"`
}
type RawQuizResponse struct {
	UserID     string `json:"user_id"`      // O ID do utilizador que pediu
	RawLLMJson string `json:"raw_llm_json"` // A string JSON crua vinda da LLM
}
