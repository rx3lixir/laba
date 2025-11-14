package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateMessage creates a new voice message record
func (s *PostgresStore) CreateMessage(ctx context.Context, msg *VoiceMessage) error {
	query := `
		INSERT INTO voice_messages (
			id, sender_id, recipient_id, file_path, file_size,
			duration_seconds, audio_format, total_chunks, chunks_received,
			status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	_, err := s.db.Exec(ctx, query,
		msg.ID,
		msg.SenderID,
		msg.RecipientID,
		msg.FilePath,
		msg.FileSize,
		msg.DurationSecs,
		msg.AudioFormat,
		msg.TotalChunks,
		msg.ChunksReceived,
		msg.Status,
		msg.CreatedAt,
	)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("opration cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetMessageByID retrieves a message by ID
func (s *PostgresStore) GetMessageByID(ctx context.Context, id uuid.UUID) (*VoiceMessage, error) {
	query := `
		SELECT
			id, sender_id, recipient_id, file_path, file_size,
			duration_seconds, audio_format, total_chunks, chunks_received,
			status, created_at, transmitted_at, delivered_at, listened_at
		FROM voice_messages
		WHERE id = $1
	`

	msg := &VoiceMessage{}
	err := s.db.QueryRow(ctx, query, id).Scan(
		&msg.ID,
		&msg.SenderID,
		&msg.RecipientID,
		&msg.FilePath,
		&msg.FileSize,
		&msg.DurationSecs,
		&msg.AudioFormat,
		&msg.TotalChunks,
		&msg.ChunksReceived,
		&msg.Status,
		&msg.CreatedAt,
		&msg.TransmittedAt,
		&msg.DeliveredAt,
		&msg.ListenedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return msg, nil
}

// GetMessagesByRecipient retrieves all messages received by a user
func (s *PostgresStore) GetMessagesByRecipient(ctx context.Context, recipientID uuid.UUID, limit, offset int) ([]*VoiceMessage, error) {
	query := `
		SELECT 
			id, sender_id, recipient_id, file_path, file_size,
			duration_seconds, audio_format, total_chunks, chunks_received,
			status, created_at, transmitted_at, delivered_at, listened_at
		FROM voice_messages
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, recipientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := []*VoiceMessage{}
	for rows.Next() {
		msg := &VoiceMessage{}
		err := rows.Scan(
			&msg.ID,
			&msg.SenderID,
			&msg.RecipientID,
			&msg.FilePath,
			&msg.FileSize,
			&msg.DurationSecs,
			&msg.AudioFormat,
			&msg.TotalChunks,
			&msg.ChunksReceived,
			&msg.Status,
			&msg.CreatedAt,
			&msg.TransmittedAt,
			&msg.DeliveredAt,
			&msg.ListenedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// UpdateMessage updates a message
func (s *PostgresStore) UpdateMessage(ctx context.Context, msg *VoiceMessage) error {
	query := `
		UPDATE voice_messages
		SET 
			chunks_received = $2,
			status = $3,
			transmitted_at = $4,
			delivered_at = $5,
			listened_at = $6
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query,
		msg.ID,
		msg.ChunksReceived,
		msg.Status,
		msg.TransmittedAt,
		msg.DeliveredAt,
		msg.ListenedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// UpdateMessageStatus updates just the status of a message
func (s *PostgresStore) UpdateMessageStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE voice_messages SET status = $2 WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// DeleteMessage deletes a message
func (s *PostgresStore) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM voice_messages WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}
