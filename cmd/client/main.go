package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/rx3lixir/laba/internal/udp"
	"github.com/rx3lixir/laba/pkg/jwt"
)

type Client struct {
	conn          *net.UDPConn
	serverAddr    *net.UDPAddr
	userID        uuid.UUID
	jwtToken      string
	authenticated bool
	logger        *log.Logger
	ackChan       chan *udp.Packet
	ctx           context.Context
	cancel        context.Context
}

func main() {
	serverAddr := flag.String("server", "localhost:9090", "UDP server address")
	jwtToken := flag.String("token", "", "JWT authentication token")
	flag.Parse()

	if *jwtToken == "" {
		fmt.Println("Error: JWT token is required")
		fmt.Println("Usage: client -token YOUR_JWT_TOKEN [-server localhost:9090]")
		os.Exit(1)
	}

	// Setup logger
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           log.InfoLevel,
	})

	// Create client
	client, err := NewClient
}
