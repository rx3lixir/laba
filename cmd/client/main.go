package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/rx3lixir/laba/internal/udp"
)

type Client struct {
	conn          *net.UDPConn
	serverAddr    *net.UDPAddr
	userID        uuid.UUID
	jwtToken      string
	authenticated bool
	logger        *log.Logger
	ackChan       chan *udp.Packet
	dataChan      chan *udp.Packet
	listChan      chan *udp.Packet
	ctx           context.Context
	cancel        context.CancelFunc

	downloadChunks map[uuid.UUID]map[uint32][]byte
	downloadTotal  map[uuid.UUID]uint32
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
	client, err := NewClient(*serverAddr, *jwtToken, logger)
	if err != nil {
		logger.Fatal("Failed to create client", "error", err)
	}
	defer client.Close()

	logger.Info("UDP Voice Chat Client started")
	logger.Info("Server address", "addr", *serverAddr)

	// Authenticate with server
	logger.Info("Authenticating...")
	if err := client.Authenticate(); err != nil {
		logger.Fatal("Authentication failed", "error", err)
	}

	logger.Info("✓ Authentication successful", "user_id", client.userID)

	// Check for messages after auth
	if err := client.CheckMessages(); err != nil {
		logger.Error("Failed to check messages", "error", err)
	}

	// Starting interactive mode if user is authenticated
	client.InteractiveMode()
}

func NewClient(serverAddr, jwtToken string, logger *log.Logger) (*Client, error) {
	// Resolve server address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:           conn,
		serverAddr:     udpAddr,
		jwtToken:       jwtToken,
		logger:         logger,
		ackChan:        make(chan *udp.Packet, 100),
		dataChan:       make(chan *udp.Packet, 100),
		listChan:       make(chan *udp.Packet, 100),
		ctx:            ctx,
		cancel:         cancel,
		downloadChunks: make(map[uuid.UUID]map[uint32][]byte),
		downloadTotal:  make(map[uuid.UUID]uint32),
	}

	// Start listening for responses
	go client.listen()

	return client, nil
}

// listen is listens idk
func (c *Client) listen() {
	buffer := make([]byte, 2048)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := c.conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				c.logger.Error("Error reading from UDP", "error", err)
				continue
			}

			// Make sure we have enough data
			if n < 48 {
				c.logger.Warn("Received packet too small", "bytes", n)
				continue
			}

			packet, err := udp.Unmarshal(buffer[:n])
			if err != nil {
				c.logger.Error("Failed to unmarshal packet", "error", err, "bytes", n)
				continue
			}
			c.handlePacket(packet)
		}
	}
}

func (c *Client) handlePacket(packet *udp.Packet) {
	switch packet.Type {
	case udp.PacketTypeAuthAck:
		c.logger.Debug("Received auth ACK")
		c.ackChan <- packet

	case udp.PacketTypeAck:
		c.logger.Debug("Received ACK",
			"message_id", packet.MessageID,
			"chunk", packet.ChunkIndex,
		)
		c.ackChan <- packet

	case udp.PacketTypeError:
		c.logger.Error("Received error from server", "error", string(packet.Payload))

	case udp.PacketTypeVoiceData:
		c.logger.Info("Received voice message",
			"message_id", packet.MessageID,
			"chunk", fmt.Sprintf("%d/%d", packet.ChunkIndex, packet.TotalChunks),
			"from", packet.SenderID,
		)
		c.dataChan <- packet

	case udp.PacketTypeMessageList:
		c.logger.Debug("Received message list")
		c.listChan <- packet

	default:
		c.logger.Warn("Unknown packet type", "type", packet.Type)
	}
}

func (c *Client) Authenticate() error {
	c.logger.Info("Authenticating with server...")

	// Create auth packet
	authPacket := udp.NewAuthPacket(uuid.Nil, c.jwtToken)

	// Send auth packet
	if err := c.sendPacket(authPacket); err != nil {
		return fmt.Errorf("failed to send auth packet: %w", err)
	}

	// Wait for ACK with timeout
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	select {
	case ack := <-c.ackChan:
		if ack.Type == udp.PacketTypeAuthAck {
			c.authenticated = true
			c.userID = ack.RecipientID // Server sends our ID back
			return nil
		}
		return fmt.Errorf("unexpected response type: %d", ack.Type)

	case <-ctx.Done():
		return fmt.Errorf("authentication timeout")
	}
}

func (c *Client) CheckMessages() error {
	if !c.authenticated {
		return fmt.Errorf("not authenticated")
	}

	c.logger.Info("Checking for messages...")

	packet := udp.NewListMessagesPacket(c.userID)
	if err := c.sendPacket(packet); err != nil {
		return fmt.Errorf("failed to send list request: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	select {
	case listPacket := <-c.listChan:
		messages, err := udp.ParseMessageList(listPacket.Payload)
		if err != nil {
			return fmt.Errorf("failed to parse message list: %w", err)
		}

		if len(messages) == 0 {
			fmt.Println("\n No unread messages")
		} else {
			fmt.Printf("\n You have %d unread message(s):\n", len(messages))
			fmt.Println(strings.Repeat("=", 70))
			for i, msg := range messages {
				fmt.Printf("%d. From: %s (%s)\n", i+1, msg.SenderName, msg.SenderID)
				fmt.Printf("   Size: %d bytes | Format: %s | Status: %s\n",
					msg.FileSize, msg.AudioFormat, msg.Status)
				fmt.Printf("   Received: %s\n", msg.CreatedAt)
				fmt.Printf("   Message ID: %s\n", msg.ID)
				fmt.Println(strings.Repeat("-", 70))
			}
			fmt.Println("Use 'download <message_id>' to download a message")
		}
		return nil

	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for message list")
	}
}

func (c *Client) DownloadMessage(messageID uuid.UUID, outputPath string) error {
	c.logger.Info("Requesting message download", "message_id", messageID)

	// Initialize chunk tracking
	c.downloadChunks[messageID] = make(map[uint32][]byte)
	c.downloadTotal[messageID] = 0

	packet := udp.NewDownloadMessagePacket(c.userID, messageID)
	if err := c.sendPacket(packet); err != nil {
		return fmt.Errorf("failed to send download request: %w", err)
	}

	// Wait for chunks
	timeout := time.After(30 * time.Second)
	var totalChunks uint32

	for {
		select {
		case dataPacket := <-c.dataChan:
			if dataPacket.MessageID != messageID {
				continue
			}

			totalChunks = dataPacket.TotalChunks
			c.downloadChunks[messageID][dataPacket.ChunkIndex] = dataPacket.Payload

			fmt.Printf("\rDownloading... %d/%d chunks",
				len(c.downloadChunks[messageID]), totalChunks)

			// Check if we have all chunks
			if uint32(len(c.downloadChunks[messageID])) == totalChunks {
				fmt.Println("\n✓ All chunks received, assembling file...")

				// Assemble file
				var assembled []byte
				for i := uint32(0); i < totalChunks; i++ {
					chunk, ok := c.downloadChunks[messageID][i]
					if !ok {
						return fmt.Errorf("missing chunk %d", i)
					}
					assembled = append(assembled, chunk...)
				}

				// Save to file
				if err := os.WriteFile(outputPath, assembled, 0o644); err != nil {
					return fmt.Errorf("failed to save file: %w", err)
				}

				// Clean up
				delete(c.downloadChunks, messageID)
				delete(c.downloadTotal, messageID)

				c.logger.Info("Message downloaded successfully",
					"path", outputPath,
					"size", len(assembled),
				)
				fmt.Printf("✓ Message saved to: %s (%d bytes)\n", outputPath, len(assembled))
				return nil
			}

		case <-timeout:
			delete(c.downloadChunks, messageID)
			delete(c.downloadTotal, messageID)
			return fmt.Errorf("download timeout")
		}
	}
}

func (c *Client) sendPacket(packet *udp.Packet) error {
	data, err := packet.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	_, err = c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

func (c *Client) sendWithRetry(packet *udp.Packet, maxRetries int) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := c.sendPacket(packet); err != nil {
			return err
		}

		// Wait for ACK
		ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)

		select {
		case ack := <-c.ackChan:
			cancel()
			if ack.MessageID == packet.MessageID && ack.ChunkIndex == packet.ChunkIndex {
				return nil
			}
		case <-ctx.Done():
			cancel()
			c.logger.Warn("ACK timeout retrying...", "attempt", attempt+1, "chunk", packet.ChunkIndex)
			continue
		}
	}

	return fmt.Errorf("max retries exceeded")
}

func (c *Client) SendVoiceMessage(recipientID uuid.UUID, filePath string) error {
	c.logger.Info("Sending voice message", "file", filePath, "to", recipientID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	c.logger.Info("File loaded", "size", len(data), "bytes")

	// Generate message ID
	messageID := uuid.New()

	// Split into chunks
	chunkSize := udp.MaxPayloadSize
	totalChunks := (len(data) + chunkSize - 1) / chunkSize

	c.logger.Info("Splitting into chunks",
		"total_chunks", totalChunks,
		"chunk_size", chunkSize,
	)

	// Send each chunk
	successfulChunks := 0
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunkData := data[start:end]

		// Create packet
		packet := udp.NewVoiceDataPacket(
			c.userID,
			recipientID,
			messageID,
			uint32(i),
			uint32(totalChunks),
			chunkData,
		)

		// Send with retry
		if err := c.sendWithRetry(packet, 3); err != nil {
			c.logger.Error("Failed to send chunk", "chunk", i, "error", err)
			continue
		}

		successfulChunks++
		c.logger.Info(
			"Chunk sent",
			"progress", fmt.Sprintf("%d/%d", i+1, totalChunks),
		)

		// To not overwhelm the network
		time.Sleep(10 * time.Millisecond)
	}

	if successfulChunks == totalChunks {
		c.logger.Info("✓ All chunks sent successfully", "message_id", messageID)
		return nil
	}

	return fmt.Errorf("only %d/%d chunks sent successfylly", successfulChunks, totalChunks)
}

func (c *Client) InteractiveMode() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n---- UDP govorilka -----")
	fmt.Println("Commands:")
	fmt.Println("send <recipient_id> <file_path>      - Send a voice message")
	fmt.Println("check                                - Check for new messages")
	fmt.Println("download <message_id> [output_path]  - Download a message")
	fmt.Println("heartbeat                            - Send heartbeat to server")
	fmt.Println("quit                                 - Exit the client")
	fmt.Println()

	for {
		fmt.Print(">_ ")
		input, err := reader.ReadString('\n')
		if err != nil {
			c.logger.Error("Error reading input", "error", err)
			continue
		}

		input = strings.TrimSpace(input)
		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		command := parts[0]

		switch command {
		case "send":
			if len(parts) != 3 {
				fmt.Println("Usage: send <recipient_id> <file_path>")
				continue
			}

			recipientID, err := uuid.Parse(parts[1])
			if err != nil {
				fmt.Println("Invalid recipient ID:", err)
				continue
			}

			filePath := parts[2]

			if err := c.SendVoiceMessage(recipientID, filePath); err != nil {
				fmt.Println("Error sending message:", err)
			}

		case "check":
			if err := c.CheckMessages(); err != nil {
				fmt.Println("Error checking messages:", err)
			}

		case "download":
			if len(parts) < 2 {
				fmt.Println("Usage: download <message_id> [output_path]")
				continue
			}

			messageID, err := uuid.Parse(parts[1])
			if err != nil {
				fmt.Println("Invalid message ID:", err)
				continue
			}

			outputPath := fmt.Sprintf("message_%s.opus", messageID.String()[:8])
			if len(parts) >= 3 {
				outputPath = parts[2]
			}

			// Ensure directory exists
			dir := filepath.Dir(outputPath)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					fmt.Println("Error creating directory:", err)
					continue
				}
			}

			if err := c.DownloadMessage(messageID, outputPath); err != nil {
				fmt.Println("Error downloading message:", err)
			}

		case "heartbeat":
			packet := udp.NewPacket(udp.PacketTypeHeartbeat, c.userID, uuid.Nil, uuid.New())
			if err := c.sendPacket(packet); err != nil {
				fmt.Println("Error sending heartbeat:", err)
			} else {
				fmt.Println("Heartbeat sent")
			}

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command:", command)
			fmt.Println("Type 'help' for available commands")
		}
	}
}

func (c *Client) Close() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
}
