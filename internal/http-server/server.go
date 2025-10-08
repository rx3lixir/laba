package httpserver

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/laba/internal/db"
)

type Server struct {
	userStorer db.UserStore
	log        *log.Logger
	httpServer *http.Server
}

func New(addr string, userStorer *db.UserStore, log *log.Logger) *Server {
	router := setupRoutes()

	return &Server{
		userStorer: *userStorer,
		log:        log,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}
