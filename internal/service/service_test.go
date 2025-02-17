package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"LinkApp/internal/storage"
)

var (
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu  sync.Mutex
)

func safeRandIntn(n int) int {
	mu.Lock()
	defer mu.Unlock()
	return rnd.Intn(n)
}

type mockStorage struct {
	sync.RWMutex
	urls map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		urls: make(map[string]string),
	}
}

func (m *mockStorage) SaveURL(ctx context.Context, shortCode, originalURL string, expiresAt time.Time) error {
	m.Lock()
	defer m.Unlock()
	m.urls[shortCode] = originalURL
	return nil
}

func (m *mockStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	m.RLock()
	defer m.RUnlock()
	if url, ok := m.urls[shortCode]; ok {
		return url, nil
	}
	return "", storage.ErrNotFound
}

type mockCache struct {
	sync.RWMutex
	data map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]string),
	}
}

func (m *mockCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	m.Lock()
	defer m.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", storage.ErrNotFound
}

func TestCreateShortURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errType error
	}{
		{
			name:    "valid url",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "invalid url",
			url:     "not-a-url",
			wantErr: true,
			errType: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewURLService(newMockStorage(), newMockCache())
			shortCode, err := s.CreateShortURL(context.Background(), tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("expected error type %v, got %v", tt.errType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(shortCode) != 6 {
				t.Errorf("expected short code length 6, got %d", len(shortCode))
			}
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		setupMock func(*mockStorage, *mockCache)
		wantURL   string
		wantErr   error
	}{
		{
			name:      "successful get from cache",
			shortCode: "cached",
			setupMock: func(ms *mockStorage, mc *mockCache) {
				mc.data["cached"] = "https://example.com/cached"
			},
			wantURL: "https://example.com/cached",
			wantErr: nil,
		},
		{
			name:      "successful get from storage",
			shortCode: "stored",
			setupMock: func(ms *mockStorage, mc *mockCache) {
				ms.urls["stored"] = "https://example.com/stored"
			},
			wantURL: "https://example.com/stored",
			wantErr: nil,
		},
		{
			name:      "not found",
			shortCode: "notfound",
			setupMock: func(ms *mockStorage, mc *mockCache) {},
			wantURL:   "",
			wantErr:   storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorage()
			mockCache := newMockCache()
			tt.setupMock(mockStorage, mockCache)

			service := NewURLService(mockStorage, mockCache)
			url, err := service.GetOriginalURL(context.Background(), tt.shortCode)

			if err != tt.wantErr {
				t.Errorf("beklenen hata %v, alınan hata %v", tt.wantErr, err)
			}

			if url != tt.wantURL {
				t.Errorf("beklenen URL %s, alınan URL %s", tt.wantURL, url)
			}
		})
	}
}

func TestCreateShortURLWithErrors(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		storage storage.Storage
		cache   storage.Cache
		wantErr error
	}{
		{
			name:    "storage error",
			url:     "https://example.com",
			storage: NewErrorStorage(fmt.Errorf("storage error")),
			cache:   newMockCache(),
			wantErr: fmt.Errorf("storage error"),
		},
		{
			name:    "cache error should not fail request",
			url:     "https://example.com",
			storage: newMockStorage(),
			cache:   NewErrorCache(fmt.Errorf("cache error")),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewURLService(tt.storage, tt.cache)
			_, err := s.CreateShortURL(context.Background(), tt.url)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) {
				t.Errorf("beklenen hata %v, alınan hata %v", tt.wantErr, err)
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("beklenen hata mesajı %v, alınan %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetOriginalURLWithErrors(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		storage   storage.Storage
		cache     storage.Cache
		wantErr   error
	}{
		{
			name:      "storage error after cache miss",
			shortCode: "test123",
			storage:   NewErrorStorage(fmt.Errorf("storage error")),
			cache:     newMockCache(),
			wantErr:   fmt.Errorf("storage error"),
		},
		{
			name:      "cache error should fallback to storage",
			shortCode: "test123",
			storage:   newMockStorage(),
			cache:     NewErrorCache(fmt.Errorf("cache error")),
			wantErr:   storage.ErrNotFound, // Storage boş olduğu için NotFound dönmeli
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewURLService(tt.storage, tt.cache)
			_, err := s.GetOriginalURL(context.Background(), tt.shortCode)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) {
				t.Errorf("beklenen hata %v, alınan hata %v", tt.wantErr, err)
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("beklenen hata mesajı %v, alınan %v", tt.wantErr, err)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	storage := newMockStorage()
	cache := newMockCache()
	service := NewURLService(storage, cache)
	ctx := context.Background()

	// Concurrent yazma işlemleri
	t.Run("concurrent writes", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()
				url := fmt.Sprintf("https://example%d.com", i)
				_, err := service.CreateShortURL(ctx, url)
				if err != nil {
					t.Errorf("concurrent write failed: %v", err)
				}
			}(i)
		}

		wg.Wait()
	})

	// Concurrent okuma işlemleri
	t.Run("concurrent reads", func(t *testing.T) {
		// Önce bir URL oluştur
		shortCode, err := service.CreateShortURL(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("URL oluşturulamadı: %v", err)
		}

		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				_, err := service.GetOriginalURL(ctx, shortCode)
				if err != nil {
					t.Errorf("concurrent read failed: %v", err)
				}
			}()
		}

		wg.Wait()
	})

	// Concurrent okuma ve yazma işlemleri
	t.Run("concurrent reads and writes", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // Okuma ve yazma için

		// Yazma goroutine'leri
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()
				url := fmt.Sprintf("https://example%d.com", i)
				shortCode, err := service.CreateShortURL(ctx, url)
				if err != nil {
					t.Errorf("concurrent write failed: %v", err)
				}
				// Her yazma işleminden sonra okuma yap
				_, err = service.GetOriginalURL(ctx, shortCode)
				if err != nil {
					t.Errorf("concurrent read after write failed: %v", err)
				}
			}(i)
		}

		// Okuma goroutine'leri
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				shortCode := fmt.Sprintf("test%d", safeRandIntn(100))
				_, _ = service.GetOriginalURL(ctx, shortCode)
			}()
		}

		wg.Wait()
	})
} 