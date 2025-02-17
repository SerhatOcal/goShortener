package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/url"
	"time"

	"LinkApp/internal/storage"
)

var (
	ErrInvalidURL = errors.New("invalid URL format")
)

type URLService struct {
	storage storage.Storage
	cache   storage.Cache
}

func NewURLService(storage storage.Storage, cache storage.Cache) *URLService {
	return &URLService{
		storage: storage,
		cache:   cache,
	}
}

func (s *URLService) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	// URL formatını kontrol et
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return "", ErrInvalidURL
	}

	// Kısa kod oluştur
	shortCode, err := generateShortCode()
	if err != nil {
		return "", err
	}

	// URL'yi veritabanına kaydet
	expiresAt := time.Now().Add(24 * time.Hour)
	err = s.storage.SaveURL(ctx, shortCode, originalURL, expiresAt)
	if err != nil {
		return "", err
	}

	// Cache'e kaydet
	err = s.cache.Set(ctx, shortCode, originalURL, 24*time.Hour)
	if err != nil {
		// Cache hatası kritik değil, sadece logla
		log.Printf("Cache error: %v", err)
	}

	return shortCode, nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Önce cache'e bak
	if url, err := s.cache.Get(ctx, shortCode); err == nil {
		return url, nil
	}

	// Cache'de yoksa veritabanından al
	url, err := s.storage.GetURL(ctx, shortCode)
	if err != nil {
		return "", err
	}

	// Bulduğumuz URL'yi cache'e ekle
	_ = s.cache.Set(ctx, shortCode, url, 24*time.Hour)

	return url, nil
}

// 6 karakterlik rastgele bir kod oluşturur
func generateShortCode() (string, error) {
	b := make([]byte, 4) // 4 byte, base64 ile kodlandığında 6 karakter olacak
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:6], nil
}
