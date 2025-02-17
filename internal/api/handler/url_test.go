package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"LinkApp/internal/service"
)

func TestCreateURL(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name: "valid request",
			requestBody: CreateURLRequest{
				URL: "https://example.com",
			},
			wantStatusCode: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name: "invalid url",
			requestBody: CreateURLRequest{
				URL: "not-a-url",
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := service.NewURLService(service.NewMockStorage(), service.NewMockCache())
			handler := NewURLHandler(s)

			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateURL(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("expected status code %d, got %d", tt.wantStatusCode, rec.Code)
			}

			var response APIResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Success != tt.wantSuccess {
				t.Errorf("expected success %v, got %v", tt.wantSuccess, response.Success)
			}
		})
	}
}

func TestGetURL(t *testing.T) {
	tests := []struct {
		name           string
		shortCode      string
		acceptHeader   string
		setupMock      func(*service.MockStorage, *service.MockCache)
		wantStatusCode int
		wantRedirect   bool
	}{
		{
			name:         "valid request - JSON response",
			shortCode:    "test123",
			acceptHeader: "application/json",
			setupMock: func(ms *service.MockStorage, mc *service.MockCache) {
				_ = ms.SaveURL(context.Background(), "test123", "https://example.com", time.Now().Add(time.Hour))
			},
			wantStatusCode: http.StatusOK,
			wantRedirect:   false,
		},
		{
			name:         "valid request - redirect",
			shortCode:    "test123",
			acceptHeader: "text/html",
			setupMock: func(ms *service.MockStorage, mc *service.MockCache) {
				_ = ms.SaveURL(context.Background(), "test123", "https://example.com", time.Now().Add(time.Hour))
			},
			wantStatusCode: http.StatusMovedPermanently,
			wantRedirect:   true,
		},
		{
			name:         "not found",
			shortCode:    "notfound",
			acceptHeader: "application/json",
			setupMock:    func(ms *service.MockStorage, mc *service.MockCache) {},
			wantStatusCode: http.StatusNotFound,
			wantRedirect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := service.NewMockStorage()
			mockCache := service.NewMockCache()
			tt.setupMock(mockStorage, mockCache)

			urlService := service.NewURLService(mockStorage, mockCache)
			handler := NewURLHandler(urlService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/urls/"+tt.shortCode, nil)
			req.Header.Set("Accept", tt.acceptHeader)
			rec := httptest.NewRecorder()

			handler.GetURL(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("beklenen status code %d, alınan %d", tt.wantStatusCode, rec.Code)
			}

			if tt.wantRedirect {
				if loc := rec.Header().Get("Location"); loc == "" {
					t.Error("yönlendirme için Location header bekleniyor")
				}
			}
		})
	}
}

func TestCreateURLWithStorageError(t *testing.T) {
	s := service.NewURLService(
		service.NewErrorStorage(fmt.Errorf("storage error")),
		service.NewMockCache(),
	)
	handler := NewURLHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls",
		strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateURL(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("beklenen status code %d, alınan %d",
			http.StatusInternalServerError, rec.Code)
	}

	var response APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("response decode edilemedi: %v", err)
	}

	if response.Success {
		t.Error("hata durumunda success false olmalı")
	}
}
