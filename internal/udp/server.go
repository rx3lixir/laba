package udp

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/rx3lixir/laba/internal/db"
	"github.com/rx3lixir/laba/internal/session"
	"github.com/rx3lixir/laba/pkg/jwt"
)

const MaxPacketSize = 2048

// Server represents a UDP server for voice messages
type Server struct {
	addr           string
	conn           *net.UDPConn
	sessionManager *session.Manager
	jwtService     *jwt.Service
	userStore      db.UserStore
	messageStore   db.MessageStore
	logger         *log.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
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

	// TODO:
	// 1. Retrieve all chunks from key-value storage
	// 2. Assemble them into a complete file
	// 3. Upload to S3
	// 4. Update database
	// 5. Forward to recipient if online
	// 6. Clean up key-value storage
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
