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
	"github.com/rx3lixir/laba/internal/session"
	"github.com/rx3lixir/laba/internal/udp"
	"github.com/rx3lixir/laba/pkg/jwt"
	"github.com/rx3lixir/laba/pkg/s3storage"
)

func main() {
	// Setting up logger
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
		Level:           log.DebugLevel,
	})

	// Initializing global context instance
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		"http_addr", c.GeneralParams.HTTPaddress,
		"udp_addr", c.UDPParams.GetAddress(),
		"database", c.MainDBParams.Name,
		"auth", c.AuthDBParams.Host,
	)

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

	logger.Info(
		"Database connection established",
		"db", c.MainDBParams.GetDSN(),
	)

	// Creates database store
	store := db.NewPostgresStore(pool)

	// Initializing JWT service
	jwtService := jwt.NewService(
		c.GeneralParams.SecretKey,
		15*time.Minute,
		7*24*time.Hour,
	)

	logger.Info("JWT service initialized")

	// Initialize Key-value storage
	sessionManager, err := session.NewManager(
		c.AuthDBParams.Host,
		c.AuthDBParams.Password,
	)
	if err != nil {
		logger.Error("Failed to create session manager", "error", err)
		os.Exit(1)
	}
	defer sessionManager.Close()

	logger.Info("Key-Value session manger initialized")

	// Initialize S3 client
	s3Client, err := s3storage.NewMinIOClient(
		c.S3Params.Endpoint,
		c.S3Params.AccessKeyID,
		c.S3Params.SecretAccessKey,
		c.S3Params.BucketName,
		c.S3Params.UseSSL,
	)
	if err != nil {
		logger.Error("Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	logger.Info("S3 storage client initialized", "bucket", c.S3Params.BucketName)

	// Creates HTTP server
	HTTPserver := httpserver.New(
		c.GeneralParams.HTTPaddress,
		store,
		jwtService,
		logger,
	)

	// Creates UDP server
	udpServer := udp.New(
		c.UDPParams.GetAddress(),
		sessionManager,
		jwtService,
		store, // UserStore
		store, // MessageStore
		s3Client,
		logger,
	)

	// Channel to listen for errors coming from the servers
	serverErrors := make(chan error, 2)

	// Start the HTTP server in a gorutine
	go func() {
		serverErrors <- HTTPserver.Start()
	}()

	// Start the UDP server in a gorutine
	go func() {
		serverErrors <- udpServer.Start()
	}()

	logger.Info("All servers started successfully")

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

		// Shutting down servers
		logger.Info("Shutting down HTTP server...")
		if err := HTTPserver.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown failed", "error", err)
		}
		logger.Info("Shutting down HTTP server...")
		if err := udpServer.Shutdown(ctx); err != nil {
			logger.Error("UDP server graceful shutdown failed", "error", err)
		}

		logger.Info("All servers stopped gracefully")
	}
}
