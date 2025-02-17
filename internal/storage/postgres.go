package storage

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := createTable(db); err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func createTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_code VARCHAR(10) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE
	)`

	_, err := db.Exec(query)
	return err
}

func (s *PostgresStorage) SaveURL(ctx context.Context, shortCode, originalURL string, expiresAt time.Time) error {
	query := `
	INSERT INTO urls (short_code, original_url, expires_at)
	VALUES ($1, $2, $3)`

	_, err := s.db.ExecContext(ctx, query, shortCode, originalURL, expiresAt)
	return err
}

func (s *PostgresStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	query := `
	SELECT original_url FROM urls 
	WHERE short_code = $1 AND (expires_at IS NULL OR expires_at > NOW())`

	var originalURL string
	err := s.db.QueryRowContext(ctx, query, shortCode).Scan(&originalURL)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return originalURL, err
} 