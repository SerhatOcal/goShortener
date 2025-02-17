package service

import (
	"context"
	"sync"
	"time"

	"LinkApp/internal/storage"
)

type MockStorage struct {
	sync.RWMutex
	urls map[string]string
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		urls: make(map[string]string),
	}
}

func (m *MockStorage) SaveURL(ctx context.Context, shortCode, originalURL string, expiresAt time.Time) error {
	m.Lock()
	defer m.Unlock()
	m.urls[shortCode] = originalURL
	return nil
}

func (m *MockStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	m.RLock()
	defer m.RUnlock()
	if url, ok := m.urls[shortCode]; ok {
		return url, nil
	}
	return "", storage.ErrNotFound
}

type MockCache struct {
	sync.RWMutex
	data map[string]string
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]string),
	}
}

func (m *MockCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	m.Lock()
	defer m.Unlock()
	m.data[key] = value
	return nil
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", storage.ErrNotFound
}

// ErrorStorage, hata durumlarını test etmek için kullanılır
type ErrorStorage struct {
	err error
}

func NewErrorStorage(err error) *ErrorStorage {
	return &ErrorStorage{err: err}
}

func (s *ErrorStorage) SaveURL(ctx context.Context, shortCode, originalURL string, expiresAt time.Time) error {
	return s.err
}

func (s *ErrorStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	return "", s.err
}

// ErrorCache, cache hatalarını test etmek için kullanılır
type ErrorCache struct {
	err error
}

func NewErrorCache(err error) *ErrorCache {
	return &ErrorCache{err: err}
}

func (c *ErrorCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	return c.err
}

func (c *ErrorCache) Get(ctx context.Context, key string) (string, error) {
	return "", c.err
}
