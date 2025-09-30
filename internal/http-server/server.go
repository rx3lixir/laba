package httpserver

import (
	"net/http"
	"time"
)

func New(addr string) *http.Server {
	router := setupRoutes()

	return &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
