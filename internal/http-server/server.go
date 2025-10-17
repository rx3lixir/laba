package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/laba/internal/db"
)

type Server struct {
	userStore  db.UserStore
	log        *log.Logger
	httpServer *http.Server
	ctx        context.Context
}

func New(addr string, userStore db.UserStore, logger *log.Logger) *Server {
	s := &Server{
		userStore: userStore,
		log:       logger,
	}

	router := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start begins listening fot HTTP requests
// This is a blocking operation
func (s *Server) Start() error {
	s.log.Info(
		"Starting HTTP server",
		"addr", s.httpServer.Addr,
	)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info(
		"Server shutting down gracefully...",
		"addr", s.httpServer.Addr,
	)
	return s.httpServer.Shutdown(ctx)
}
