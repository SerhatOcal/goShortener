package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"LinkApp/internal/api/handler"
	"LinkApp/internal/service"
	"LinkApp/internal/storage"
)

func main() {
	log.Println("URL Shortener Service starting...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

    pgConnStr := "postgresql://almadansorgula:15825270Se@localhost:5432/urlshortener?sslmode=disable"

	// Redis bağlantı bilgileri
	redisAddr := fmt.Sprintf("%s:%s",
		"localhost",
		"6379",
	)

	// PostgreSQL storage'ı oluştur
	log.Println("Connecting to PostgreSQL...")
	pg, err := storage.NewPostgresStorage(pgConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Redis cache'i oluştur
	log.Println("Connecting to Redis...")
	redis, err := storage.NewRedisCache(redisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// URL servisini oluştur
	urlService := service.NewURLService(pg, redis)

	// HTTP handler'ı oluştur
	urlHandler := handler.NewURLHandler(urlService)

	// HTTP router'ı ayarla
	mux := http.NewServeMux()

	// API endpoint'lerini ayarla
	mux.HandleFunc("/api/v1/urls", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			urlHandler.CreateURL(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/urls/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if strings.TrimPrefix(r.URL.Path, "/api/v1/urls/") != "" {
				urlHandler.GetURL(w, r)
				return
			}
			http.NotFound(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "web/index.html")
			return
		}
		http.NotFound(w, r)
	})

	// HTTP sunucusunu oluştur
	server := &http.Server{
		Addr:    ":9090",
		Handler: mux,
	}

	// HTTP sunucusunu başlat
	go func() {
		log.Printf("HTTP server starting on :9090")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown için sinyal yakalama
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Service is ready")

	// Servis kapanana kadar bekle
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v\n", sig)
		cancel()
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
	}

	log.Println("Shutting down gracefully...")
}
