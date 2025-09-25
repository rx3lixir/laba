package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Route() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Privet Android"))
	})
}
