package s3storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps the MinIO client for voice message storage
type MinIOClient struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOClient creates a new MinIO client and ensures bucket exists
func NewMinIOClient(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	mc := &MinIOClient{
		client:     client,
		bucketName: bucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mc.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return mc, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (m *MinIOClient) ensureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// UploadVoiceMessage uploads a voice message file to MinIO
// Returns the object path in MinIO
func (m *MinIOClient) UploadVoiceMessage(
	ctx context.Context,
	messageID uuid.UUID,
	data []byte,
	audioFormat string,
) (string, error) {
	// Format: messages/YYYY/MM/DD/messageID.format
	now := time.Now()
	objectName := fmt.Sprintf(
		"messages/%d/%02d/%02d/%s.%s",
		now.Year(),
		now.Day(),
		now.Month(),
		messageID.String(),
		audioFormat,
	)

	// Determine content type based on format
	contentType := "audio/opus"
	switch audioFormat {
	case "mp3":
		contentType = "audio/mpeg"
	case "ogg":
		contentType = "audio/ogg"
	case "wav":
		contentType = "audio/wav"
	}

	// Upload the file
	reader := bytes.NewReader(data)
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		objectName,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to minio: %w", err)
	}

	return objectName, nil
}

// DownloadVoiceMessage downloads a voice message from MinIO
func (m *MinIOClient) DownloadVoiceMessage(ctx context.Context, objectName string) ([]byte, error) {
	object, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// DeleteVoiceMessage deletes a voice message from MinIO
func (m *MinIOClient) DeleteVoiceMessage(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// DeleteVoiceMessage deletes a voice message from MinIO
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}
	return url.String(), nil
}

// GetObjectInfo retrieves metadata about a stored object
func (m *MinIOClient) GetObjectInfo(ctx context.Context, objectName string) (*minio.ObjectInfo, error) {
	info, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}
	return &info, nil
}
