package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vadimbarashkov/medods-test-task/internal/service"

	v1 "github.com/vadimbarashkov/medods-test-task/internal/http/handler/v1"
)

func NewRouter(logger *slog.Logger, authService *service.AuthService) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(SlogLogger(logger))
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		v1.RegisterAuthRoutes(r, authService)
	})

	return r
}
