package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"LinkApp/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

type CreateURLRequest struct {
	URL string `json:"url"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type CreateURLResponse struct {
	ShortCode   string    `json:"short_code"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	OriginalURL string    `json:"original_url"`
}

type GetURLResponse struct {
	OriginalURL string    `json:"original_url"`
	ShortCode   string    `json:"short_code"`
	AccessedAt  time.Time `json:"accessed_at"`
}

func (h *URLHandler) CreateURL(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	shortCode, err := h.service.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			sendErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		default:
			sendErrorResponse(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	resp := CreateURLResponse{
		ShortCode:   shortCode,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		OriginalURL: req.URL,
	}

	sendSuccessResponse(w, http.StatusCreated, resp)
}

func (h *URLHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	shortCode := strings.TrimPrefix(r.URL.Path, "/api/v1/urls/")
	if shortCode == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Short code is required")
		return
	}

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortCode)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			sendErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		default:
			sendErrorResponse(w, http.StatusNotFound, "URL not found")
		}
		return
	}

	// API isteği mi yoksa tarayıcı isteği mi kontrol et
	if r.Header.Get("Accept") == "application/json" {
		resp := GetURLResponse{
			OriginalURL: originalURL,
			ShortCode:   shortCode,
			AccessedAt:  time.Now(),
		}
		sendSuccessResponse(w, http.StatusOK, resp)
		return
	}

	// Tarayıcı isteği ise yönlendir
	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func sendErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

func sendSuccessResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}
