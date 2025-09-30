package db

import (
	"context"
	"fmt"
	"time"
)

func addContext(parentCtx context.Context, timeout int) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*time.Duration(timeout))
	defer cancel()

	return ctx, cancel
}

func (s *PostgresStore) CreateUser(parentCtx context.Context, user *User) error {
	ctx, cancel := addContext(parentCtx, 3)
	defer cancel()

	query := `
			INSERT INTO users (name, email, password)
			VALUES ($1, $2, $3)
			RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.Password,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("Failed to create user: %w", err)
	}

	return nil
}
