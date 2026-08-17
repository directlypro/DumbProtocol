package http_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type server struct {
	httpServer *http.Server
	url        string
	timeout    time.Duration
	router     *chi.Mux
	// add TOTP API
}

func NewServer(url string, timeout time.Duration) (*server, error) {
	addr := strings.TrimPrefix(url, "http://")
	addr = strings.TrimPrefix(addr, "https://")

	router := chi.NewRouter()
	s := &server{
		url:     addr,
		timeout: timeout,
		router:  router,
	}
	return s, nil
}

func (s *server) Start() error {
	if s == nil || s.router == nil {
		return errors.New("server is not initialized")
	}

	s.router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Compress(5),
		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.RealIP,
		middleware.RequestID,
		middleware.Timeout(s.timeout),
	)

	s.routes()

	s.httpServer = &http.Server{
		Addr:    s.url,
		Handler: s.router,
	}

	return s.httpServer.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("server is not initialized")
	}
	fmt.Println("Shutting down HTTP server")

	return s.httpServer.Shutdown(ctx)
}

func (s *server) routes() {
	s.router.Get("/healthz", s.handleHealthCheck)
}

func (s *server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

