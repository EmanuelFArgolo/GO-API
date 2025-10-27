package store

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// Store wraps access to the database.
type Store struct {
	DB *sql.DB
}

// NewPostgresStore initializes a new Store backed by Postgres using the provided
// connection string (lib/pq format, e.g., "host=... port=... user=... password=... dbname=... sslmode=disable").
func NewPostgresStore(connStr string) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Reasonable pool settings for a small service; adjust as needed
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{DB: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}
