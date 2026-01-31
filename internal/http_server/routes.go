package http_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type server struct {
	httpServer *http.Server
	url        string
	timeout    time.Duration
	router     *chi.Mux
	// add TOPT API
}

func NewServer(url string, timeout time.Duration) (*server, error) {
	s := &server{
		url:     url,
		timeout: timeout,
	}
	return s, nil
}

func (s *server) Start() error {
	if s == nil {
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
	if s == nil {
		return errors.New("server is not initialized")
	}
	fmt.Println("Shutting down HTTP server")

	return s.httpServer.Shutdown(ctx)
}

func (s *server) routes() {
	// add all relevant routes here
}
