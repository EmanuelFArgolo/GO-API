// internal/store/submission_repo.go
package store

import (
	"context"
	"fmt"
	"log"
	"quizz-core/internal/models"
	"strconv"
)

// CorrectAnswerInfo é uma struct simples para guardar o gabarito
type CorrectAnswerInfo struct {
	QuestionID      int    `db:"id"`
	QuestionAssunto string `db:"assunto"`
	CorrectOption   string `db:"corpo"` // O texto da resposta correta
}

// GetQuizAnswers busca o gabarito (respostas corretas) para um quiz
func (s *Store) GetQuizAnswers(ctx context.Context, quizID int) (map[string]CorrectAnswerInfo, error) {
	// Esta query junta perguntas e respostas, filtrando apenas as respostas corretas
	// e apenas para o quiz_id especificado.
	query := `
		SELECT p.id, p.assunto, r.corpo
		FROM perguntas p
		JOIN respostas r ON p.id = r.pergunta_id
		WHERE p.quizz_id = $1 AND r.correta = TRUE
	`

	var answers []CorrectAnswerInfo
	if err := s.DB.SelectContext(ctx, &answers, query, quizID); err != nil {
		return nil, fmt.Errorf("falha ao buscar gabarito: %w", err)
	}

	answerMap := make(map[string]CorrectAnswerInfo)
	for _, ans := range answers {
		answerMap[strconv.Itoa(ans.QuestionID)] = ans
	}

	return answerMap, nil
}

// SaveSubmissionStats é uma função transacional para salvar os resultados completos
func (s *Store) SaveSubmissionStats(ctx context.Context, sub models.Submissao, dadas []models.RespostaDada, difs []models.Dificuldade) (*models.Submissao, error) {

	// 1. Iniciar a transação
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao iniciar transação de submissão: %w", err)
	}
	defer tx.Rollback()

	// 2. Etapa A: Salvar a Submissão principal (tabela 'submissao')
	var savedSub models.Submissao
	err = tx.GetContext(ctx, &savedSub,
		`INSERT INTO submissao (datahora, pontuacao, utilizador_id, quizz_id)
		 VALUES ($1, $2, $3, $4) RETURNING *`,
		sub.DataHora, sub.Pontuacao, sub.UtilizadorID, sub.QuizzID,
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao inserir submissao: %w", err)
	}

	// Usamos o ID da submissão que acabamos de criar
	submissionID := savedSub.ID

	// 3. Etapa B: Salvar as Respostas Dadas (tabela 'respostas_dadas')
	for _, dada := range dadas {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO respostas_dadas (submissao_id, pergunta_id, resposta_id, correta_na_submissao)
			 VALUES ($1, $2, $3, $4)`,
			submissionID, dada.PerguntaID, nil, dada.CorretaNaSubmissao, // Ignoramos resposta_id por enquanto
		)
		if err != nil {
			return nil, fmt.Errorf("falha ao inserir resposta_dada: %w", err)
		}
	}

	// 4. Etapa C: Salvar as Dificuldades (tabela 'dificuldades')
	for _, dif := range difs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO dificuldades (assunto, submissao_id)
			 VALUES ($1, $2)`,
			dif.Assunto, submissionID,
		)
		if err != nil {
			return nil, fmt.Errorf("falha ao inserir dificuldade: %w", err)
		}
	}

	// 5. Finalizar a Transação
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("falha ao commitar transação de submissão: %w", err)
	}

	log.Printf("Submissão %d salva com sucesso (%d respostas dadas, %d dificuldades).",
		savedSub.ID, len(dadas), len(difs))

	return &savedSub, nil
}
