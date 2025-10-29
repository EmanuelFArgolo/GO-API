// internal/store/theme_repo.go
package store

import (
	"context"
	"fmt"
	"quizz-core/internal/models"
)

// GetAllActiveThemes busca todos os temas que estão marcados como 'ativo = TRUE'
func (s *Store) GetAllActiveThemes(ctx context.Context) ([]models.Tema, error) {

	query := "SELECT * FROM tema WHERE ativo = TRUE ORDER BY nome ASC"

	temas := []models.Tema{}
	if err := s.DB.SelectContext(ctx, &temas, query); err != nil {
		// Se não houver linhas, apenas retornamos uma lista vazia, não um erro
		
		return nil, fmt.Errorf("falha ao buscar temas ativos: %w", err)
	}

	return temas, nil
}
