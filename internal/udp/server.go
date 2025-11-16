package udp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/rx3lixir/laba/internal/db"
	"github.com/rx3lixir/laba/internal/session"
	"github.com/rx3lixir/laba/pkg/jwt"
	"github.com/rx3lixir/laba/pkg/s3storage"
)

const MaxPacketSize = 2048

// Server represents a UDP server for voice messages
type Server struct {
	addr            string
	conn            *net.UDPConn
	sessionManager  *session.Manager
	jwtService      *jwt.Service
	userStore       db.UserStore
	messageStore    db.MessageStore
	s3storageClient *s3storage.MinIOClient
	logger          *log.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// New creates a new UDP server
func New(
	addr string,
	sessionMgr *session.Manager,
	jwtSvc *jwt.Service,
	userStore db.UserStore,
	messageStore db.MessageStore,
	logger *log.Logger,
) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		addr:           addr,
		sessionManager: sessionMgr,
		jwtService:     jwtSvc,
		userStore:      userStore,
		messageStore:   messageStore,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the UDP server
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	s.conn = conn
	s.logger.Info("UDP server started", "address", s.addr)

	s.wg.Add(1)
	go s.listen()

	return nil
}

func (s *Server) listen() {
	defer s.wg.Done()

	buffer := make([]byte, MaxPacketSize)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("UDP server stopping...")
			return
		default:
			n, clientAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				s.logger.Error("Error reading from UDP", "error", err)
				continue
			}

			// Process packet in a goroutine to noe block receiving
			packetData := make([]byte, n)
			copy(packetData, buffer[:n])

			s.wg.Add(1)
			go s.handlePacket(packetData, clientAddr)
		}
	}
}

func (s *Server) handlePacket(data []byte, clientAddr *net.UDPAddr) {
	defer s.wg.Done()

	packet, err := Unmarshal(data)
	if err != nil {
		s.logger.Error("Failed to unmarshal packet", "error", err, "from", clientAddr)
		return
	}

	s.logger.Debug(
		"Received packet",
		"type", packet.Type,
		"from", clientAddr,
		"message_id", packet.MessageID,
		"chunk", fmt.Sprintf("%d/%d", packet.ChunkIndex, packet.TotalChunks),
	)

	// Handle diefferent packet types
	switch packet.Type {
	case PacketTypeAuth:
		s.handleAuth(packet, clientAddr)
	case PacketTypeVoiceData:
		s.handleVoiceData(packet, clientAddr)
	case PacketTypeHeartbeat:
		s.handleHeartbeat(packet, clientAddr)
	default:
		s.logger.Warn("Unknown packet type", "type", packet.Type, "from", clientAddr)
	}
}

// handleAuth proccesses authentication UDP packets
func (s *Server) handleAuth(packet *Packet, clientAddr *net.UDPAddr) {
	jwtToken := string(packet.Payload)

	claims, err := s.jwtService.ValidateToken(jwtToken)
	if err != nil {
		s.logger.Warn("Invalid JWT in auth packet", "error", err, "from", clientAddr)
		s.sendErrorPacket(clientAddr, packet.MessageID, "Invalid token")
		return
	}

	// Create session
	err = s.sessionManager.CreateSession(s.ctx, claims.UserID, claims.Username, clientAddr)
	if err != nil {
		s.logger.Error("Failed to create session", "error", err, "user_id", claims.UserID)
		s.sendErrorPacket(clientAddr, packet.MessageID, "Failed to create session")
		return
	}

	s.logger.Info(
		"User authenticated",
		"user_id", claims.UserID,
		"username", claims.Username,
		"address", clientAddr,
	)

	ackPacket := NewAckPacket(packet)
	s.sendPacket(ackPacket, clientAddr)
}

// handleVoiceData proccesses voice data chunks
func (s *Server) handleVoiceData(packet *Packet, clientAddr *net.UDPAddr) {
	session, err := s.sessionManager.GetSession(s.ctx, packet.SenderID)
	if err != nil {
		s.logger.Warn("Packet from unauthenticated user", "sender_id", packet.SenderID)
		return
	}

	s.sessionManager.UpdateLastSeen(s.ctx, packet.SenderID)

	// Saving current chunk to keyvalue storage
	err = s.sessionManager.SavePendingChunk(s.ctx, packet.MessageID, packet.ChunkIndex, packet.Payload)
	if err != nil {
		s.logger.Error("Failed to save a chunk", "error", err, "message_id", packet.MessageID)
		return
	}

	// Increment chunk counter
	count, err := s.sessionManager.IncrementChunksReceived(s.ctx, packet.MessageID)
	if err != nil {
		s.logger.Error("Failed to increment chunk counter", "error", err)
		return
	}

	s.logger.Debug(
		"Chunk received",
		"message_id", packet.MessageID,
		"chunk", fmt.Sprintf("%d/%d", packet.ChunkIndex, packet.TotalChunks),
		"total_received", count,
		"from", session.Username,
	)

	// Send ACK
	ackPacket := NewAckPacket(packet)
	s.sendPacket(ackPacket, clientAddr)

	// Check if all chunks received
	if uint32(count) == packet.TotalChunks {
		s.logger.Info("All chunks received", "message_id", packet.MessageID, "total", packet.TotalChunks)
		s.wg.Add(1)
		go s.processCompleteMessage(packet.MessageID, packet.SenderID, packet.RecipientID, packet.TotalChunks)
	}
}

// processCompleteMessage assembles chunks and save the complete file
func (s *Server) processCompleteMessage(messageID uuid.UUID, senderID, recipientID uuid.UUID, totalChunks uint32) {
	defer s.wg.Done()

	s.logger.Info("Proccessing complete message", "message_id", messageID)

	// 1. Retrieve all chunks from key-val storage
	chunks := make([][]byte, totalChunks)
	var totalSize int

	for i := uint32(0); i < totalChunks; i++ {
		chunkData, err := s.sessionManager.GetPendingChunk(s.ctx, messageID, i)
		if err != nil {
			s.logger.Error(
				"Failed to retrieve chunk",
				"message_id", messageID,
				"chunk", i,
				"error", err,
			)
			s.updateMessageStatus(messageID, db.MessageStatusFailed)
			return
		}
		chunks[i] = chunkData
		totalSize += len(chunkData)
	}

	// 2. Assemble chunks into complete file
	assembledData := make([]byte, 0, totalSize)
	for _, chunk := range chunks {
		assembledData = append(assembledData, chunk...)
	}

	s.logger.Info("File assembled", "message_id", messageID, "size", len(assembledData))

	// 3. Upload to s3 storage
	audioFromat := "opus" // default

	objectPath, err := s.s3storageClient.UploadVoiceMessage(s.ctx, messageID, assembledData, audioFromat)
	if err != nil {
		s.logger.Error(
			"Failed to upload to s3",
			"message_id", messageID,
			assembledData,
			audioFromat,
		)
	}

	// 4. Create database record
	now := time.Now()
	voiceMessage := &db.VoiceMessage{
		ID:             messageID,
		SenderID:       senderID,
		RecipientID:    recipientID,
		FilePath:       objectPath,
		FileSize:       len(assembledData),
		AudioFormat:    audioFromat,
		TotalChunks:    int(totalChunks),
		ChunksReceived: int(totalChunks),
		Status:         db.MessageStatusTransmitted,
		TransmittedAt:  &now,
	}

	if err := s.messageStore.CreateMessage(s.ctx, voiceMessage); err != nil {
		s.logger.Error("Failed to create message record", "message_id", messageID, "error", err)
		// Still mark as transmitted as file is in s3
	} else {
		s.logger.Info("Message record created", "message_id", messageID)
	}

	// 5. Forward to recipient if online
	recipientOnline, err := s.sessionManager.IsUserOnline(s.ctx, recipientID)
	if err != nil {
		s.logger.Warn(
			"Failed to check recipient status",
			"recipient_id", recipientID,
			"error", err,
		)
	} else if recipientOnline {
		s.logger.Info(
			"Recipient is online, forwarding message",
			"recipient_id", recipientID,
		)
		s.forwardMessageToRecipient(messageID, senderID, recipientID, assembledData, totalChunks)
	} else {
		s.logger.Info(
			"Recipient is offline, message stored for later retrieval",
			"recipient_id", recipientID,
		)

		// 6. Clean up key-value storage
		if err := s.sessionManager.DeletePendingMessage(s.ctx, messageID, totalChunks); err != nil {
			s.logger.Warn("Failed to clean up pending message", "message_id", messageID, "error", err)
		} else {
			s.logger.Info("Pending message cleaned up", "message_id", messageID)
		}

		s.logger.Info("✓ Message processing complete", "message_id", messageID)
	}
}

// forwardMessageToRecipient sends the message to an online recipient
func (s *Server) forwardMessageToRecipient(messageID uuid.UUID, senderID, recipientID uuid.UUID, data []byte, totalChunks uint32) {
	// Get recipient session to find their UDP address
	recipientSession, err := s.sessionManager.GetSession(s.ctx, recipientID)
	if err != nil {
		s.logger.Error("Failed to get recipient session", "recipient_id", recipientID, "error", err)
		return
	}

	// Parse recipient UDP address
	recipientAddr, err := net.ResolveUDPAddr("udp", recipientSession.Address)
	if err != nil {
		s.logger.Error(
			"Failed to resolve recipient address",
			"address", recipientSession.Address,
			"error", err,
		)
		return
	}

	s.logger.Info(
		"Forwarding message to recipient",
		"recipient", recipientSession.Username,
		"address", recipientAddr,
		"chunks", totalChunks,
	)

	// Split back into chunks and send
	chunkSize := MaxPayloadSize

	for i := uint32(0); i < totalChunks; i++ {
		start := int(i) * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunkData := data[start:end]

		packet := NewVoiceDataPacket(senderID, recipientID, messageID, i, totalChunks, chunkData)
		s.sendPacket(packet, recipientAddr)

		time.Sleep(5 * time.Millisecond)
	}

	s.logger.Info(
		"Message forwarded successfully",
		"message_id", messageID,
		"recipient", recipientSession.Username,
	)

	now := time.Now()
	msg := &db.VoiceMessage{
		ID:          messageID,
		Status:      db.MessageStatusDelivered,
		DeliveredAt: &now,
	}
	s.messageStore.UpdateMessage(s.ctx, msg)
}

// handleHeartbeat keeps the session alive
func (s *Server) handleHeartbeat(packet *Packet, clientAddr *net.UDPAddr) {
	err := s.sessionManager.UpdateLastSeen(s.ctx, packet.SenderID)
	if err != nil {
		s.logger.Warn("Heartbeat from unknown user", "sender_id", packet.SenderID)
		return
	}

	ackPacket := NewAckPacket(packet)
	s.sendPacket(ackPacket, clientAddr)
}

// updateMessageStatus is a helper to update message status
func (s *Server) updateMessageStatus(messageId uuid.UUID, status string) {
	if err := s.messageStore.UpdateMessageStatus(s.ctx, messageId, status); err != nil {
		s.logger.Error(
			"Failed to update message status",
			"message_id", messageId,
			"status", status,
			"error", err,
		)
	}
}

// sendPacket sends a packet to a client
func (s *Server) sendPacket(packet *Packet, addr *net.UDPAddr) {
	data, err := packet.Marshal()
	if err != nil {
		s.logger.Error("Failed to marshal packet", "error", err)
		return
	}

	_, err = s.conn.WriteToUDP(data, addr)
	if err != nil {
		s.logger.Error("Failed to send packet", "error", err, "to", addr)
	}
}

// sendErrorPacket sends an error UDP packet
func (s *Server) sendErrorPacket(addr *net.UDPAddr, messageID uuid.UUID, errorMsg string) {
	packet := NewPacket(PacketTypeError, uuid.Nil, uuid.Nil, messageID)
	packet.Payload = []byte(errorMsg)
	s.sendPacket(packet, addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down UDP server...")

	s.cancel()

	// Close the connection
	if s.conn != nil {
		s.conn.Close()
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("UDP server shut down gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("UDP server shutdown timeout")
		return ctx.Err()
	}
}
