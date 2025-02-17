package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("url not found")
)

type Storage interface {
	SaveURL(ctx context.Context, shortCode, originalURL string, expiresAt time.Time) error
	GetURL(ctx context.Context, shortCode string) (string, error)
}

type Cache interface {
	Set(ctx context.Context, key, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
} 