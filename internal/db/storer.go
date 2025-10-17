package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	sqlc "github.com/rx3lixir/laba/internal/db/sqlc"
)

// PostgresStore embeds the sqlc Queries
type PostgresStore struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// NewPostgresStore creates a new store with the connection pool
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		queries: sqlc.New(pool),
		pool:    pool,
	}
}

// CreatePostgresPool creates and pings a connection pool
func CreatePostgresPool(parentCtx context.Context, dburl string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	pool, err := pgxpool.New(ctx, dburl)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
