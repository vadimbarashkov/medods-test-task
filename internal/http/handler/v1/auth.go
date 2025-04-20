package v1

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/vadimbarashkov/medods-test-task/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func RegisterAuthRoutes(r chi.Router, authService *service.AuthService) {
	h := &AuthHandler{authService: authService}

	r.Post("/auth/tokens", h.IssueTokens)
	r.Post("/auth/tokens/refresh", h.RefreshTokens)
}

type TokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *AuthHandler) IssueTokens(w http.ResponseWriter, r *http.Request) {
	userIDParam := r.URL.Query().Get("user_id")
	if userIDParam == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error: "missing user_id query param",
		})
		return
	}

	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error: "invalid user_id query param",
		})
		return
	}

	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	accessToken, refreshToken, err := h.authService.IssueTokens(r.Context(), userID, net.ParseIP(clientIP))
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error: "failed to issue tokens",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, TokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header not found")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

func (h *AuthHandler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := extractBearerToken(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, ErrorResponse{
			Error: "missing or invalid authorization header",
		})
		return
	}

	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	accessToken, newRefreshToken, err := h.authService.RefreshTokens(r.Context(), refreshToken, net.ParseIP(clientIP))
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, ErrorResponse{
				Error: "invalid refresh token",
			})
			return
		}
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error: "failed to refresh tokens",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, TokensResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}
