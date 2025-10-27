// internal/store/stats_repo.go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"quizz-core/internal/models"
	"strconv"
)

// statsDBRow é uma struct interna para ler os resultados agregados do SQL
type statsDBRow struct {
	TotalQuizzesRealizados    sql.NullInt64   `db:"total_quizzes"`
	PontuacaoMedia            sql.NullFloat64 `db:"avg_score"`
	TotalPerguntasRespondidas sql.NullInt64   `db:"total_respostas"`
	TotalAcertos              sql.NullInt64   `db:"total_acertos"`
}

// GetUserStats calcula as estatísticas agregadas para um utilizador
func (s *Store) GetUserStats(ctx context.Context, userID int) (*models.UserStatsResponse, error) {

	// Esta query junta 'submissao' e 'respostas_dadas' para calcular tudo de uma vez
	query := `
		SELECT
			COUNT(DISTINCT s.id) AS total_quizzes,
			AVG(s.pontuacao) AS avg_score,
			COUNT(rd.id) AS total_respostas,
			SUM(CASE WHEN rd.correta_na_submissao = TRUE THEN 1 ELSE 0 END) AS total_acertos
		FROM
			submissao s
		LEFT JOIN
			respostas_dadas rd ON s.id = rd.submissao_id
		WHERE
			s.utilizador_id = $1
	`

	var stats statsDBRow
	if err := s.DB.GetContext(ctx, &stats, query, userID); err != nil {
		if err == sql.ErrNoRows {
			// Se não houver linhas, significa que o utilizador existe mas nunca fez um quiz
			// Retornamos estatísticas "zero" em vez de um erro
			return &models.UserStatsResponse{
				UserID: strconv.Itoa(userID), // Precisamos de strconv
			}, nil
		}
		return nil, fmt.Errorf("falha ao buscar estatísticas do utilizador %d: %w", userID, err)
	}

	// Converter os tipos do DB (que podem ser nulos) para os tipos do nosso modelo

	totalErros := stats.TotalPerguntasRespondidas.Int64 - stats.TotalAcertos.Int64
	var percentagemAcerto float64 = 0
	if stats.TotalPerguntasRespondidas.Int64 > 0 {
		percentagemAcerto = (float64(stats.TotalAcertos.Int64) / float64(stats.TotalPerguntasRespondidas.Int64)) * 100.0
	}

	response := &models.UserStatsResponse{
		UserID:                    strconv.Itoa(userID),
		TotalQuizzesRealizados:    int(stats.TotalQuizzesRealizados.Int64),
		PontuacaoMedia:            stats.PontuacaoMedia.Float64,
		TotalPerguntasRespondidas: int(stats.TotalPerguntasRespondidas.Int64),
		TotalAcertos:              int(stats.TotalAcertos.Int64),
		TotalErros:                int(totalErros),
		PercentagemAcerto:         percentagemAcerto,
	}

	return response, nil
}
