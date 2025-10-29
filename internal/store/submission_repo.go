// internal/store/submission_repo.go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"quizz-core/internal/models"
	"strconv"
	"time"
)

// Question answer info é uma struct simples para guardar o gabarito
type QuestionAnswerInfo struct {
	QuestionID        int
	Assunto           string
	CorrectOptionText string         // O texto da opção correta
	OptionsMap        map[string]int // Mapa de [Texto da Opção] -> [ID da Opção]
}

// dbAnswerRow é uma struct interna para ler todas as opcoes
type dbAnswerRow struct {
	QuestionID      int     `db:"pergunta_id"`
	QuestionAssunto *string `db:"assunto"`
	RespostaID      int     `db:"resposta_id"`
	CorpoResposta   string  `db:"corpo_resposta"`
	Correta         bool    `db:"correta"`
}

// GetQuizAnswers busca o gabarito (respostas corretas) para um quiz
func (s *Store) GetQuizAnswers(ctx context.Context, quizID int) (map[string]QuestionAnswerInfo, error) {
	// Esta query busca TODAS as respostas de TODAS as perguntas de um quiz
	query := `
		SELECT
			p.id AS pergunta_id,
			p.assunto,
			r.id AS resposta_id,
			r.corpo AS corpo_resposta,
			r.correta
		FROM
			perguntas p
		JOIN
			respostas r ON p.id = r.pergunta_id
		WHERE
			p.quizz_id = $1
	`

	var allAnswers []dbAnswerRow
	if err := s.DB.SelectContext(ctx, &allAnswers, query, quizID); err != nil {
		return nil, fmt.Errorf("falha ao buscar gabarito completo: %w", err)
	}

	// Agora, processamos o resultado (que está "achatado") num mapa complexo
	answerMap := make(map[string]QuestionAnswerInfo)

	for _, row := range allAnswers {
		qIDStr := strconv.Itoa(row.QuestionID)

		// Verifica se já começámos a processar esta pergunta
		info, exists := answerMap[qIDStr]
		if !exists {
			// Primeira vez que vemos esta pergunta
			info = QuestionAnswerInfo{
				QuestionID: row.QuestionID,
				Assunto:    "", // Default
				OptionsMap: make(map[string]int),
			}
			if row.QuestionAssunto != nil {
				info.Assunto = *row.QuestionAssunto
			}
		}

		// Adiciona a opção ao mapa de opções
		info.OptionsMap[row.CorpoResposta] = row.RespostaID

		// Se esta for a resposta correta, guarda o texto
		if row.Correta {
			info.CorrectOptionText = row.CorpoResposta
		}

		// Coloca de volta no mapa
		answerMap[qIDStr] = info
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
			submissionID, dada.PerguntaID, dada.RespostaID, dada.CorretaNaSubmissao, // <-- O 'dada.RespostaID' é a correção
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

type submissionDetailDBRow struct {
	// Info da Submissão e Quiz (repetido em cada linha)
	SubmissionID int       `db:"submissao_id"`
	QuizID       int       `db:"quizz_id"`
	QuizNome     string    `db:"quiz_nome"`
	TemaNome     string    `db:"tema_nome"`
	Pontuacao    float64   `db:"pontuacao"`
	DataHora     time.Time `db:"datahora"`

	// Info da Pergunta
	PerguntaID    int     `db:"pergunta_id"`
	CorpoPergunta string  `db:"corpo_pergunta"`
	Assunto       *string `db:"assunto"`

	// Info da Resposta (Opção)
	RespostaID    int    `db:"resposta_id"`
	CorpoResposta string `db:"corpo_resposta"`
	Correta       bool   `db:"correta"` // Se *esta opção* é a correta

	// Info da Resposta Dada pelo Utilizador
	// Usamos sql.NullInt64 e sql.NullBool porque pode não haver resposta dada para uma pergunta
	RespostaDadaID     sql.NullInt64 `db:"resposta_dada_id"`     // O ID da resposta que o user escolheu
	CorretaNaSubmissao sql.NullBool  `db:"correta_na_submissao"` // Se o user acertou esta pergunta
}

// GetSubmissionDetails busca todos os detalhes de uma submissão específica
func (s *Store) GetSubmissionDetails(ctx context.Context, submissionID int) (*models.SubmissionDetailResponse, error) {

	// Query complexa que junta 5 tabelas!
	query := `
		SELECT
			s.id AS submissao_id,
			s.quizz_id,
			q.nome AS quiz_nome,
			t.nome AS tema_nome,
			s.pontuacao,
			s.datahora,
			p.id AS pergunta_id,
			p.corpo AS corpo_pergunta,
			p.assunto,
			r.id AS resposta_id,
			r.corpo AS corpo_resposta,
			r.correta,
			rd.resposta_id AS resposta_dada_id,
			rd.correta_na_submissao
		FROM
			submissao s
		JOIN
			quizzes q ON s.quizz_id = q.id
		JOIN
			tema t ON q.tema_id = t.id
		JOIN
			perguntas p ON q.id = p.quizz_id
		JOIN
			respostas r ON p.id = r.pergunta_id
		LEFT JOIN -- LEFT JOIN porque pode não haver uma resposta dada
			respostas_dadas rd ON s.id = rd.submissao_id AND p.id = rd.pergunta_id
		WHERE
			s.id = $1
		ORDER BY
			p.id, r.id -- Importante ordenar para agrupar corretamente
	`

	var resultsDB []submissionDetailDBRow
	if err := s.DB.SelectContext(ctx, &resultsDB, query, submissionID); err != nil {
		if err == sql.ErrNoRows {
			// Submissão não encontrada
			return nil, sql.ErrNoRows // Retornamos o erro original para o service tratar como 404
		}
		return nil, fmt.Errorf("falha ao buscar detalhes da submissão %d: %w", submissionID, err)
	}

	// Se não houver resultados, significa que a submissão não existe
	if len(resultsDB) == 0 {
		return nil, sql.ErrNoRows
	}

	// --- Processar os Resultados (Agrupar por Pergunta) ---
	// A query retorna uma linha para CADA OPÇÃO de CADA PERGUNTA.
	// Precisamos de agrupar isto na estrutura da nossa API.

	// Pegamos a informação geral da primeira linha (é repetida)
	firstRow := resultsDB[0]
	responseAPI := &models.SubmissionDetailResponse{
		SubmissionID: firstRow.SubmissionID,
		QuizID:       firstRow.QuizID,
		QuizNome:     firstRow.QuizNome,
		TemaNome:     firstRow.TemaNome,
		Pontuacao:    firstRow.Pontuacao,
		DataHora:     firstRow.DataHora,
		Perguntas:    []models.QuestionDetailResponse{}, // Inicializa o array vazio
	}

	// Usamos um mapa para agrupar as opções por pergunta_id
	perguntasMap := make(map[int]*models.QuestionDetailResponse)

	for _, row := range resultsDB {
		perguntaID := row.PerguntaID

		// Verifica se já começámos a processar esta pergunta
		detalhePergunta, exists := perguntasMap[perguntaID]
		if !exists {
			// Primeira vez que vemos esta pergunta, criamos a struct base
			detalhePergunta = &models.QuestionDetailResponse{
				PerguntaID:    perguntaID,
				CorpoPergunta: row.CorpoPergunta,
				Assunto:       row.Assunto,
				Opcoes:        []models.AnswerOptionDetail{},
				Acertou:       row.CorretaNaSubmissao.Bool, // Pega o valor (pode ser false se NullBool)
			}
			perguntasMap[perguntaID] = detalhePergunta
		}

		// Adiciona a opção atual à lista de opções da pergunta
		detalhePergunta.Opcoes = append(detalhePergunta.Opcoes, models.AnswerOptionDetail{
			RespostaID: row.RespostaID,
			Corpo:      row.CorpoResposta,
		})

		// Se *esta opção* for a correta, guardamos o texto dela
		if row.Correta {
			detalhePergunta.RespostaCorreta = row.CorpoResposta
		}

		// Se *esta opção* foi a que o utilizador escolheu, guardamos o texto dela
		// Comparamos o ID desta opção (row.RespostaID) com o ID que o user escolheu (row.RespostaDadaID)
		if row.RespostaDadaID.Valid && row.RespostaDadaID.Int64 == int64(row.RespostaID) {
			respostaUser := row.CorpoResposta // Guarda o texto da opção escolhida
			detalhePergunta.RespostaUtilizador = &respostaUser
		}
	}

	// Adiciona as perguntas agrupadas (do mapa) à resposta final
	for _, p := range perguntasMap {
		responseAPI.Perguntas = append(responseAPI.Perguntas, *p)
	}

	return responseAPI, nil
}
