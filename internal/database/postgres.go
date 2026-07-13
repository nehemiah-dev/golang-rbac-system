package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewConn only creates and verifies the database connection pool.
func NewConn(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Don't leak the pool if ping fails
		return nil, err
	}

	return pool, nil
}
