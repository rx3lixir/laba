package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/laba/internal/config"
	"github.com/rx3lixir/laba/internal/db"
	"github.com/rx3lixir/laba/internal/http-server"
)

func main() {
	// Setting up logger
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
		Level:           log.DebugLevel,
	})

	// Initializing config manager
	cm, err := config.NewConfigManager("internal/config/config.yaml")
	if err != nil {
		logger.Error("Error getting config file", "error", err)
		os.Exit(1)
	}

	c := cm.GetConfig()

	// Validating configuration
	if err := c.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	logger.Info(
		"Configuration loaded",
		"env", c.GeneralParams.Env,
		"database", c.MainDBParams.Name,
		"auth", c.AuthDBParams.Host,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Creating database connection pool
	pool, err := db.CreatePostgresPool(ctx, c.MainDBParams.GetDSN())
	if err != nil {
		logger.Error(
			"Failed to create postgres pool",
			"error", err,
			"db", c.MainDBParams.Name,
		)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("Database connection established",
		"db", c.MainDBParams.GetDSN(),
	)

	// Creates database store
	store := db.NewPostgresStore(pool)

	// Creates HTTP server
	HTTPserver := httpserver.New(c.GeneralParams.HTTPaddress, store, logger)

	// Channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Start the server in a gorutine
	go func() {
		serverErrors <- HTTPserver.Start()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we recieve a signal or error
	select {
	case err := <-serverErrors:
		logger.Error("Server error", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("Shutdown signal received", "signal", sig)

		// Give outstanding requests 10s to complete
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutting down a server
		if err := HTTPserver.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		logger.Info("Server stopped gracefully")
	}
}
