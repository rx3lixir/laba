-- +goose Up
-- +goose StatementBegin
CREATE TABLE voice_messages (
  id UUID PRIMARY KEY,
  sender_id UUID NOT NULL,
  recipient_id UUID NOT NULL,

  -- File metadata
  file_path TEXT NOT NULL,
  file_size INTEGER NOT NULL,
  duration_seconds INTEGER,
  audio_format VARCHAR(10) DEFAULT 'opus',

  -- Transmission metadata
  total_chunks INTEGER NOT NULL,
  chunks_received INTEGER DEFAULT 0,

  -- Status: pending / transmitting / delivered / listened / failed
  status VARCHAR(20) DEFAULT 'pending',

  -- Timestamp
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  transmitted_at TIMESTAMP,
  delivered_at TIMESTAMP,
  listened_at TIMESTAMP,

  CONSTRAINT fk_sender FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_recipient FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_voice_messages_created_at ON voice_messages(created_at DESC);
CREATE INDEX idx_voice_messages_status ON voice_messages(status);
CREATE INDEX idx_voice_messages_recipient ON voice_messages(recipient_id);
CREATE INDEX idx_voice_messages_sender ON voice_messages(sender_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_voice_messages_created_at;
DROP INDEX IF EXISTS idx_voice_messages_status;
DROP INDEX IF EXISTS idx_voice_messages_recipient;
DROP INDEX IF EXISTS idx_voice_messages_sender;
DROP TABLE IF EXISTS voice_messages;
-- +goose StatementEnd
