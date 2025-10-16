package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware block
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Compress(5))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Say hi
		r.Get("/hello", s.HandleHello)

		// User routes
		r.Route("/user", func(r chi.Router) {
			r.Post("/", s.HandleAddUser)
		})
	})

	return r
}
