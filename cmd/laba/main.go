package main

import (
	"context"
	//"net/http"
	"os"
	"time"

	httpserver "github.com/rx3lixir/laba/internal/http-server"
	"github.com/rx3lixir/laba/pkg/logger"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger.Init("dev")
	defer logger.Close()

	log := logger.NewLogger()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	log.Info("Connecting to database...")

	log.Info("Successfully connected to user db")

	messageconn, err := pgx.Connect(ctx, "postgresql://admin:12345@localhost:5433/message-db")
	if err != nil {
		log.Error("Oops... failed to connect to message db", "error:", err)
		os.Exit(1)
	}
	defer messageconn.Close(ctx)

	log.Info("Successfully connected to message db")

	if err := messageconn.Ping(ctx); err != nil {
		log.Error("Oops... failed to ping message database", "error:", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	status := rdb.Ping(ctx)

	log.Info("Redis status", "status", status)

	log.Info("Db pinged")

	srv := httpserver.New("localhost:8080")

	srv.Start()

	os.Exit(0)
}
