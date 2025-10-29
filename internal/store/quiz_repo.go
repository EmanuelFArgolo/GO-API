// internal/store/quiz_repo.go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"quizz-core/internal/llm"    // Precisamos do llm.LLMQuestionResponse
	"quizz-core/internal/models" // Precisamos do models.CreateQuizRequest
)

// SaveGeneratedQuiz usa uma transação para salvar um quiz completo
// (Tema, Quiz, Perguntas e Respostas) gerado pela LLM.
func (s *Store) SaveGeneratedQuiz(ctx context.Context, req models.CreateQuizRequest, llmQuestions []llm.LLMQuestionResponse) (*models.Quiz, []models.Pergunta, error) {

	// 1. Iniciar a transação
	// Usamos Tx para garantir que todas as queries sejam executadas ou nenhuma
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	// Garante que, se algo falhar, a transação seja revertida
	defer tx.Rollback() // Rollback é ignorado se tx.Commit() for chamado

	// --- Lógica do Banco de Dados ---

	// 2. Etapa A: Encontrar ou Criar o Tema
	// Usamos o 'Theme' (ex: "Biologia Celular") do pedido
	var tema models.Tema
	// Tenta buscar o tema pelo nome
	err = tx.GetContext(ctx, &tema, "SELECT * FROM tema WHERE nome = $1 AND ativo = TRUE", req.Theme)

	if err == sql.ErrNoRows {
		// Tema não existe, vamos criá-lo
		log.Printf("Tema '%s' não encontrado, criando...", req.Theme)
		// O 'RETURNING *' nos devolve o objeto 'tema' completo (incluindo o ID e 'criacao')
		err = tx.GetContext(ctx, &tema,
			"INSERT INTO tema (nome) VALUES ($1) RETURNING *",
			req.Theme,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao inserir novo tema: %w", err)
		}
	} else if err != nil {
		// Outro erro ao buscar o tema
		return nil, nil, fmt.Errorf("falha ao buscar tema: %w", err)
	}

	// 3. Etapa B: Criar o Quiz
	// (Assumimos que sempre criamos um novo quiz)
	var quiz models.Quiz
	// O nome do quiz pode ser o próprio tema, ou um nome customizado
	quizName := req.Theme // Por enquanto, o nome do quiz é o nome do tema
	err = tx.GetContext(ctx, &quiz,
		"INSERT INTO quizzes (nome, tema_id) VALUES ($1, $2) RETURNING *",
		quizName, tema.ID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao inserir quiz: %w", err)
	}

	// 4. Etapa C e D: Criar Perguntas e Respostas (Loop)
	var perguntasSalvas []models.Pergunta

	for _, llmQ := range llmQuestions {
		// Etapa C: Inserir a Pergunta
		var pergunta models.Pergunta
		err = tx.GetContext(ctx, &pergunta,
			`INSERT INTO perguntas (assunto, corpo, explicacao, quizz_id)
			 VALUES ($1, $2, $3, $4) RETURNING *`,
			llmQ.Subject, llmQ.QuestionText, nil, quiz.ID, // (Explicacao é nil por enquanto)
		)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao inserir pergunta: %w", err)
		}

		// Etapa D: Inserir as 4 Respostas
		for _, opt := range llmQ.Options {
			isCorrect := (opt == llmQ.CorrectAnswer)
			_, err = tx.ExecContext(ctx,
				`INSERT INTO respostas (corpo, correta, pergunta_id)
				 VALUES ($1, $2, $3)`,
				opt, isCorrect, pergunta.ID,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("falha ao inserir resposta: %w", err)
			}
		}
		perguntasSalvas = append(perguntasSalvas, pergunta)
	}

	// 5. Finalizar a Transação
	// Se chegamos aqui sem erros, 'Commit' salva tudo no banco
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("falha ao commitar transação: %w", err)
	}

	log.Printf("Quiz %d salvo com sucesso (Tema ID: %d, %d Perguntas).", quiz.ID, tema.ID, len(perguntasSalvas))

	// Retornamos os objetos criados
	return &quiz, perguntasSalvas, nil
}

// DeactivateQuiz faz um "soft-delete" de um quiz, definindo ativo = FALSE
func (s *Store) DeactivateQuiz(ctx context.Context, quizID int) (int64, error) {

	query := "UPDATE quizzes SET ativo = FALSE WHERE id = $1 AND ativo = TRUE"

	// Usamos ExecContext para UPDATEs
	result, err := s.DB.ExecContext(ctx, query, quizID)
	if err != nil {
		return 0, fmt.Errorf("falha ao executar update para desativar quiz: %w", err)
	}

	// Verificamos quantas linhas foram realmente alteradas
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("falha ao verificar linhas afetadas: %w", err)
	}

	return rowsAffected, nil
}

func (s *Store) GetActiveQuizzesByTheme(ctx context.Context, themeID int) ([]models.Quiz, error) {

	query := "SELECT * FROM quizzes WHERE tema_id = $1 AND ativo = TRUE ORDER BY nome ASC"

	quizzes := []models.Quiz{}
	if err := s.DB.SelectContext(ctx, &quizzes, query, themeID); err != nil {
		
		return nil, fmt.Errorf("falha ao buscar quizzes por tema: %w", err)
	}

	return quizzes, nil
}
