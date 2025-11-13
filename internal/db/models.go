package db

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VoiceMessage struct {
	ID             uuid.UUID  `json:"id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	RecipientID    uuid.UUID  `json:"recipient_id"`
	FilePath       string     `json:"file_path"`
	FileSize       int        `json:"file_size"`
	DurationSecs   *int       `json:"duration_seconds,omitempty"`
	AudioFormat    string     `json:"audio_format"`
	TotalChunks    int        `json:"total_chunks"`
	ChunksReceived int        `json:"chunks_received"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	TransmittedAt  *time.Time `json:"transmitted_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ListenedAt     *time.Time `json:"listened_at,omitempty"`
}

const (
	MessageStatusPending     = "pending"
	MessageStatusTransmitted = "transmitted"
	MessageStatusDelivered   = "delivered"
	MessageStatusListened    = "listened"
	MessageStatusFailed      = "failed"
)
