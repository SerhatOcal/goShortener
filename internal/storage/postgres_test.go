package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestPostgresStorage(t *testing.T) {
	// Test için geçici bir veritabanı bağlantısı
	connStr := "host=localhost port=5432 user=postgres password=password dbname=urlshortener sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skip("PostgreSQL bağlantısı kurulamadı, testi atlıyorum:", err)
	}
	defer db.Close()

	// Tabloyu oluştur
	err = createTable(db)
	if err != nil {
		t.Fatal("Tablo oluşturulamadı:", err)
	}

	storage := &PostgresStorage{db: db}
	ctx := context.Background()

	t.Run("SaveURL and GetURL", func(t *testing.T) {
		shortCode := fmt.Sprintf("t%d", time.Now().UnixNano()%10000)
		originalURL := "https://example.com"
		expiresAt := time.Now().Add(24 * time.Hour)

		// URL'yi kaydet
		err := storage.SaveURL(ctx, shortCode, originalURL, expiresAt)
		if err != nil {
			t.Fatal("URL kaydedilemedi:", err)
		}

		// URL'yi getir
		got, err := storage.GetURL(ctx, shortCode)
		if err != nil {
			t.Fatal("URL getirilemedi:", err)
		}

		if got != originalURL {
			t.Errorf("Beklenen URL %s, alınan URL %s", originalURL, got)
		}
	})

	t.Run("GetURL - Not Found", func(t *testing.T) {
		_, err := storage.GetURL(ctx, "nonexistent")
		if err != ErrNotFound {
			t.Errorf("Var olmayan URL için ErrNotFound bekleniyor, alınan: %v", err)
		}
	})

	t.Run("GetURL - Expired URL", func(t *testing.T) {
		shortCode := fmt.Sprintf("e%d", time.Now().UnixNano()%10000)
		originalURL := "https://example.com/expired"
		expiresAt := time.Now().Add(-1 * time.Hour)

		// Süresi dolmuş URL'yi kaydet
		err := storage.SaveURL(ctx, shortCode, originalURL, expiresAt)
		if err != nil {
			t.Fatal("URL kaydedilemedi:", err)
		}

		// Süresi dolmuş URL'yi getirmeye çalış
		_, err = storage.GetURL(ctx, shortCode)
		if err != ErrNotFound {
			t.Errorf("Süresi dolmuş URL için ErrNotFound bekleniyor, alınan: %v", err)
		}
	})
}

func TestConcurrentStorageAccess(t *testing.T) {
	// Test için geçici bir veritabanı bağlantısı
	connStr := "host=localhost port=5432 user=postgres password=password dbname=urlshortener sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skip("PostgreSQL bağlantısı kurulamadı, testi atlıyorum:", err)
	}
	defer db.Close()

	storage := &PostgresStorage{db: db}
	ctx := context.Background()

	// Concurrent okuma işlemleri
	t.Run("concurrent reads", func(t *testing.T) {
		shortCode := fmt.Sprintf("r%d", time.Now().UnixNano()%10000)
		url := "https://example.com"
		expiresAt := time.Now().Add(24 * time.Hour)
		err := storage.SaveURL(ctx, shortCode, url, expiresAt)
		if err != nil {
			t.Fatalf("URL oluşturulamadı: %v", err)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := storage.GetURL(ctx, shortCode)
				if err != nil {
					errChan <- err
				}
			}()
		}

		// Wait for all goroutines
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("concurrent read failed: %v", err)
		}
	})

	t.Run("concurrent writes", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()
				shortCode := fmt.Sprintf("c%d_%d", i, time.Now().UnixNano()%10000)
				url := fmt.Sprintf("https://example%d.com", i)
				expiresAt := time.Now().Add(24 * time.Hour)

				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					errChan <- fmt.Errorf("transaction başlatılamadı: %v", err)
					return
				}
				defer tx.Rollback()

				err = storage.SaveURL(ctx, shortCode, url, expiresAt)
				if err != nil {
					if !strings.Contains(err.Error(), "duplicate key value") {
						errChan <- fmt.Errorf("concurrent write failed: %v", err)
					}
					return
				}

				if err := tx.Commit(); err != nil {
					errChan <- fmt.Errorf("transaction commit edilemedi: %v", err)
				}
			}(i)
		}

		// Wait for all goroutines
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Error(err)
		}
	})
} 