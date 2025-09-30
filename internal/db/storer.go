package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// To abstract db methods from pgxpool api
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, argumesnts ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresStore struct {
	db DBTX
}

func NewPostgresStore(pool DBTX) *PostgresStore {
	return &PostgresStore{
		db: pool,
	}
}

type UserStore interface {
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	GetUsers(ctx context.Context) ([]*User, error)
	GetUserByID(ctx context.Context, id int) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id int) error
}

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
