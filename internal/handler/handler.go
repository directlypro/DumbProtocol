package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"DumbProtocol/internal/config"
	"DumbProtocol/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Server struct {
	httpServer *http.Server
	config     *config.Config
	router     *chi.Mux
}

func NewServer(cfg *config.Config, totpService service.TOTPService) (*Server, error) {
	router := chi.NewRouter()

	healthH := NewHealthHandler()
	totpH := NewTOTPHandler(totpService)

	s := &Server{
		config: cfg,
		router: router,
	}

	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Compress(5),
		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.RealIP,
		middleware.RequestID,
		middleware.Timeout(cfg.ReadTimeout),
	)

	// Route definition
	router.Get("/healthz", healthH.HealthCheck)

	router.Route("/api/v1/totp", func(r chi.Router) {
		r.Post("/setup", totpH.Setup)
		r.Post("/verify", totpH.Verify)
		r.Post("/recovery", totpH.Recovery)
	})

	return s, nil
}

func (s *Server) Start() error {
	if s == nil || s.router == nil {
		return errors.New("server is not initialized")
	}

	s.httpServer = &http.Server{
		Addr:         s.config.ServerAddress(),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) RouterHandler() http.Handler {
	return s.router
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("server is not initialized")
	}
	fmt.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
