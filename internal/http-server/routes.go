package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware block
	// r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Compress(5))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/hello", s.HandleHello)

		// Public auth routes (no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", s.HandleSignup)
			r.Post("/signin", s.HandleSignin)
			r.Post("/refresh", s.HandleRefreshToken)
		})

		// Protected user routes (auth required)
		r.Route("/user", func(r chi.Router) {
			r.Use(s.AuthMiddleware)

			r.Get("/", s.HandleGetAllUsers)
			r.Get("/email/{email}", s.HandleGetUserByEmail)
			r.Get("/{id}", s.HandleGetUserByID)
			r.Post("/", s.HandleCreateUser)
			r.Delete("/{id}", s.HandleDeleteUser)
		})
	})

	return r
}
