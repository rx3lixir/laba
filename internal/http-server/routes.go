package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Compress(5))

	r.Mount("/api", apiRoutes())

	return r
}

func apiRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/hello", makeHTTPHandlerFunc(handleHello))

	r.Post("/adduser", makeHTTPHandlerFunc(handleAddUser))

	return r
}
