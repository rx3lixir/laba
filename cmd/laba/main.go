package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rx3lixir/laba/pkg/logger"
)

func main() {
	logger.Init("dev")
	defer logger.Close()

	log := logger.NewLogger()

	log.Info("Govno")

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Privet Android"))
	})

	http.ListenAndServe(":3333", r)
}
