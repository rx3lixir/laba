package udp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/google/uuid"
)

const (
	PacketTypeAuth         = 0x01
	PacketTypeAuthAck      = 0x02
	PacketTypeVoiceData    = 0x03
	PacketTypeAck          = 0x04
	PacketTypeHeartbeat    = 0x05
	PacketTypeListMessages = 0x06 // NEW: Request list of messages
	PacketTypeMessageList  = 0x07 // NEW: Response with message list
	PacketTypeDownloadMsg  = 0x08 // NEW: Request to download a message
	PacketTypeError        = 0xFF
)

const (
	ProtocolVersion = 0x01
	MaxPayloadSize  = 1400
)

// MessageInfo represents metadata about a voice message
type MessageInfo struct {
	ID          uuid.UUID `json:"id"`
	SenderID    uuid.UUID `json:"sender_id"`
	SenderName  string    `json:"sender_name"`
	FileSize    int       `json:"file_size"`
	Duration    *int      `json:"duration,omitempty"`
	AudioFormat string    `json:"audio_format"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
}

// Packet represents a UDP packet
type Packet struct {
	Version     uint8
	Type        uint8
	MessageID   uuid.UUID
	ChunkIndex  uint32
	TotalChunks uint32
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Checksum    uint32
	PayloadLen  uint16
	Payload     []byte
}

// Marshal converts a Packet to bytes
func (p *Packet) Marshal() ([]byte, error) {
	if len(p.Payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload size %d exceeds maximum %d", len(p.Payload), MaxPayloadSize)
	}

	buf := new(bytes.Buffer)

	// Version
	if err := binary.Write(buf, binary.BigEndian, p.Version); err != nil {
		return nil, err
	}

	// Type
	if err := binary.Write(buf, binary.BigEndian, p.Type); err != nil {
		return nil, err
	}

	// MessageID
	if _, err := buf.Write(p.MessageID[:]); err != nil {
		return nil, err
	}

	// ChunkIndex
	if err := binary.Write(buf, binary.BigEndian, p.ChunkIndex); err != nil {
		return nil, err
	}

	// TotalChunks
	if err := binary.Write(buf, binary.BigEndian, p.TotalChunks); err != nil {
		return nil, err
	}

	// SenderID
	if _, err := buf.Write(p.SenderID[:]); err != nil {
		return nil, err
	}

	// RecipientID
	if _, err := buf.Write(p.RecipientID[:]); err != nil {
		return nil, err
	}

	// Calculate checksum of payload
	p.Checksum = crc32.ChecksumIEEE(p.Payload)
	if err := binary.Write(buf, binary.BigEndian, p.Checksum); err != nil {
		return nil, err
	}

	// Write payload length and payload
	p.PayloadLen = uint16(len(p.Payload))
	if err := binary.Write(buf, binary.BigEndian, p.PayloadLen); err != nil {
		return nil, err
	}

	if _, err := buf.Write(p.Payload); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal converts bytes to a Packet
func Unmarshal(data []byte) (*Packet, error) {
	if len(data) < 48 {
		return nil, fmt.Errorf("packet too small: %d bytes", len(data))
	}

	buf := bytes.NewReader(data)
	p := &Packet{}

	// Read header fields
	if err := binary.Read(buf, binary.BigEndian, &p.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Type); err != nil {
		return nil, err
	}

	// MessageID
	messageIDBytes := make([]byte, 16)
	if _, err := buf.Read(messageIDBytes); err != nil {
		return nil, err
	}
	p.MessageID, _ = uuid.FromBytes(messageIDBytes)

	// ChunkIndex
	if err := binary.Read(buf, binary.BigEndian, &p.ChunkIndex); err != nil {
		return nil, err
	}

	// TotalChunks
	if err := binary.Read(buf, binary.BigEndian, &p.TotalChunks); err != nil {
		return nil, err
	}

	// SenderID
	senderIDBytes := make([]byte, 16)
	if _, err := buf.Read(senderIDBytes); err != nil {
		return nil, err
	}
	p.SenderID, _ = uuid.FromBytes(senderIDBytes)

	// RecipientID
	recipientIDBytes := make([]byte, 16)
	if _, err := buf.Read(recipientIDBytes); err != nil {
		return nil, err
	}
	p.RecipientID, _ = uuid.FromBytes(recipientIDBytes)

	// Checksum
	if err := binary.Read(buf, binary.BigEndian, &p.Checksum); err != nil {
		return nil, err
	}

	// PayloadLen
	if err := binary.Read(buf, binary.BigEndian, &p.PayloadLen); err != nil {
		return nil, err
	}

	// Read payload (only if there is one)
	if p.PayloadLen > 0 {
		p.Payload = make([]byte, p.PayloadLen)
		if _, err := buf.Read(p.Payload); err != nil {
			return nil, err
		}

		// Verify checksum
		calculatedChecksum := crc32.ChecksumIEEE(p.Payload)
		if calculatedChecksum != p.Checksum {
			return nil, fmt.Errorf("checksum mismatch: expected %d, got %d", p.Checksum, calculatedChecksum)
		}
	} else {
		p.Payload = []byte{}
	}

	return p, nil
}

// NewPacket creates a new Packet with default values
func NewPacket(packetType uint8, senderID, recipientID, messageID uuid.UUID) *Packet {
	return &Packet{
		Version:     ProtocolVersion,
		Type:        packetType,
		MessageID:   messageID,
		SenderID:    senderID,
		RecipientID: recipientID,
	}
}

// NewAuthPacket creates an authentication packet
func NewAuthPacket(userID uuid.UUID, jwtToken string) *Packet {
	p := NewPacket(PacketTypeAuth, userID, uuid.Nil, uuid.New())
	p.Payload = []byte(jwtToken)
	return p
}

// NewAckPacket creates an acknowledgment packet
func NewAckPacket(originalPacket *Packet) *Packet {
	p := NewPacket(PacketTypeAck, originalPacket.RecipientID, originalPacket.SenderID, originalPacket.MessageID)
	p.ChunkIndex = originalPacket.ChunkIndex
	p.TotalChunks = originalPacket.TotalChunks
	return p
}

// NewVoiceDataPacket creates a voice data packet
func NewVoiceDataPacket(senderID, recipientID, messageID uuid.UUID, chunkIndex, totalChunks uint32, data []byte) *Packet {
	p := NewPacket(PacketTypeVoiceData, senderID, recipientID, messageID)
	p.ChunkIndex = chunkIndex
	p.TotalChunks = totalChunks
	p.Payload = data
	return p
}

// NewListMessagesPacket creates a packet requesting message list
func NewListMessagesPacket(userID uuid.UUID) *Packet {
	return NewPacket(PacketTypeListMessages, userID, uuid.Nil, uuid.New())
}

// NewMessageListPacket creates a packet with message list response
func NewMessageListPacket(recipientID uuid.UUID, messages []MessageInfo) (*Packet, error) {
	data, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	p := NewPacket(PacketTypeMessageList, uuid.Nil, recipientID, uuid.New())
	p.Payload = data
	return p, nil
}

// NewDownloadMessagePacket creates a packet requesting message download
func NewDownloadMessagePacket(userID, messageID uuid.UUID) *Packet {
	p := NewPacket(PacketTypeDownloadMsg, userID, uuid.Nil, messageID)
	p.Payload = []byte("download") // Need payload to avoid EOF
	return p
}

// ParseMessageList parses message list from packet payload
func ParseMessageList(payload []byte) ([]MessageInfo, error) {
	var messages []MessageInfo
	if err := json.Unmarshal(payload, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}
	return messages, nil
}
