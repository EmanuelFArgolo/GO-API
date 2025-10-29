// internal/store/store.go
package store

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx" // <--- ADICIONE ESTE
	_ "github.com/lib/pq"
)

// Store agora usa sqlx.DB
type Store struct {
	DB *sqlx.DB // <--- MUDANÇA AQUI (de sql.DB para sqlx.DB)
}

// NewPostgresStore cria uma nova conexão com o banco de dados
func NewPostgresStore(connStr string) (*Store, error) {
	// Usamos sqlx.Open em vez de sql.Open
	db, err := sqlx.Open("postgres", connStr) // <--- MUDANÇA AQUI
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexão sqlx: %w", err)
	}

	// Pinga o banco de dados para garantir que a conexão está viva
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha ao pingar banco de dados: %w", err)
	}

	log.Println("Conectado ao banco de dados com sucesso! (usando sqlx)")

	return &Store{
		DB: db,
	}, nil
}
func (s *Store) Ping(ctx context.Context) error {
	// Usamos PingContext para respeitar timeouts
	return s.DB.PingContext(ctx)
}
