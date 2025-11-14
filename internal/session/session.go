package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

// Session represents a user's UDP session
type Session struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Address   string    `json:"address"`
	LastSeen  time.Time `json:"last_seen"`
	Status    string    `json:"status"`
	ConnectAt time.Time `json:"connected_at"`
}

// PendingMessage tracks chunks being received
type PendingMessage struct {
	MessageID      uuid.UUID         `json:"message_id"`
	SenderID       uuid.UUID         `json:"sender_id"`
	RecipientID    uuid.UUID         `json:"recipient_id"`
	TotalChunks    uint32            `json:"total_chunks"`
	ChunksReceived map[uint32][]byte `json:"chunks_received"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Manager handles key-value storage operations for sessions
type Manager struct {
	client valkey.Client
}

// NewManager creates a new session manager
func NewManager(addr, password string) (*Manager, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
		Password:    password,
		// DisableCache: false, // Enable client-side caching for better performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to valkey: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pingCmd := client.B().Ping().Build()
	if err := client.Do(ctx, pingCmd).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping valkey: %w", err)
	}

	return &Manager{client: client}, nil
}

func (m *Manager) CreateSession(ctx context.Context, userID uuid.UUID, username string, addr *net.UDPAddr) error {
	session := Session{
		UserID:    userID,
		Username:  username,
		Address:   addr.String(),
		LastSeen:  time.Now(),
		Status:    "online",
		ConnectAt: time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("session:%s", userID.String())

	setCmd := m.client.B().Set().
		Key(key).
		Value(string(data)).
		Ex(300). // 5 minutes = 300 seconds
		Build()

	if err := m.client.Do(ctx, setCmd).Error(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	saddCmd := m.client.B().Sadd().
		Key("online_users").
		Member(userID.String()).
		Build()

	if err := m.client.Do(ctx, saddCmd).Error(); err != nil {
		return fmt.Errorf("failed to add to online users: %w", err)
	}

	return nil
}

// GetSession retrieves a users's session
func (m *Manager) GetSession(ctx context.Context, userID uuid.UUID) (*Session, error) {
	key := fmt.Sprintf("session:%s", userID.String())

	getCmd := m.client.B().Get().Key(key).Build()

	result := m.client.Do(ctx, getCmd)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	data, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// UpdateLastSeen updates the last seen timestmp
func (m *Manager) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	session, err := m.GetSession(ctx, userID)
	if err != nil {
		return err
	}

	session.LastSeen = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", userID.String())

	// Set with updated data and reset expiration
	setCmd := m.client.B().Set().
		Key(key).
		Value(string(data)).
		Ex(300).
		Build()

	return m.client.Do(ctx, setCmd).Error()
}

// DeleteSession removes a users's session
func (m *Manager) DeleteSession(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("session:%s", userID.String())

	delCmd := m.client.B().Del().Key(key).Build()

	if err := m.client.Do(ctx, delCmd).Error(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from online users
	sremCmd := m.client.B().Srem().
		Key("online_users").
		Member(userID.String()).
		Build()

	if err := m.client.Do(ctx, sremCmd).Error(); err != nil {
		return fmt.Errorf("failed to remove from online users: %w", err)
	}

	return nil
}

// IsUserOnline checks if a user is online
func (m *Manager) IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	sismemberCmd := m.client.B().Sismember().
		Key("online_users").
		Member(userID.String()).
		Build()

	result := m.client.Do(ctx, sismemberCmd)

	// returns 1 if member exists, 0 if not
	val, err := result.AsInt64()
	if err != nil {
		return false, fmt.Errorf("failed to check online status: %w", err)
	}

	return val == 1, nil
}

// SavePendingChunk saves a received
func (m *Manager) SavePendingChunk(ctx context.Context, messageID uuid.UUID, chunkIndex uint32, data []byte) error {
	key := fmt.Sprintf("pending_message:%s:chunk:%d", messageID.String(), chunkIndex)

	setCmd := m.client.B().Set().
		Key(key).
		Value(valkey.BinaryString(data)).
		Ex(600). // 10 minutes
		Build()

	return m.client.Do(ctx, setCmd).Error()
}

// GetPendingChunk retrieves a chunk
func (m *Manager) GetPendingChunk(ctx context.Context, messageID uuid.UUID, chunkIndex uint32) ([]byte, error) {
	key := fmt.Sprintf("pending_message:%s:chunk:%d", messageID.String(), chunkIndex)

	getCmd := m.client.B().Get().Key(key).Build()

	result := m.client.Do(ctx, getCmd)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("chunk not found")
		}
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	str, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to parse chunk data: %w", err)
	}

	return []byte(str), nil
}

// IncrementChunksReceived increments the chunk counter
func (m *Manager) IncrementChunksReceived(ctx context.Context, messageID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("pending_message:%s:count", messageID.String())

	incrCmd := m.client.B().Incr().Key(key).Build()

	result := m.client.Do(ctx, incrCmd)

	count, err := result.AsInt64()
	if err != nil {
		return 0, fmt.Errorf("failed to increment chunks: %w", err)
	}

	return count, nil
}

// GetChunksReceivedCount getts the current chunk count
func (m *Manager) GetChunksReceivedCount(ctx context.Context, messageID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("pending_message:%s:count", messageID.String())

	getCmd := m.client.B().Get().Key(key).Build()

	result := m.client.Do(ctx, getCmd)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return 0, nil
		}

		return 0, fmt.Errorf("failed to get chunks count: %w", err)
	}

	count, err := result.AsInt64()
	if err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return count, nil
}

// DeletePendingMessage removes all pending message data
func (m *Manager) DeletePendingMessage(ctx context.Context, messageID uuid.UUID, totalChunks uint32) error {
	keys := make([]string, 0, totalChunks+1)

	// Add all chunk keys
	for i := uint32(0); i < totalChunks; i++ {
		key := fmt.Sprintf("pending_message:%s:chunk:%d", messageID.String(), i)
		keys = append(keys, key)
	}

	// Add the counter key
	countKey := fmt.Sprintf("pending_message:%s:count", messageID.String())
	keys = append(keys, countKey)

	delCmd := m.client.B().Del().Key(keys...).Build()

	return m.client.Do(ctx, delCmd).Error()
}

// Close closes the client connection
func (m *Manager) Close() {
	m.client.Close()
}
