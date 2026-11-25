package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// PoolConfig configures PostgreSQL connection pooling for Bun.
type PoolConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open creates a Bun PostgreSQL database handle with bounded pool settings.
func Open(cfg PoolConfig) (*bun.DB, error) {
	if cfg.DSN == "" {
		return nil, errors.New("database: DSN is required")
	}
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DSN)))

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle < 0 || maxIdle > maxOpen {
		maxIdle = maxOpen / 2
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = 30 * time.Minute
	}

	sqldb.SetMaxOpenConns(maxOpen)
	sqldb.SetMaxIdleConns(maxIdle)
	sqldb.SetConnMaxLifetime(lifetime)

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}

// Ping verifies PostgreSQL connectivity within the provided context deadline.
func Ping(ctx context.Context, db *bun.DB) error {
	if db == nil {
		return errors.New("database: db handle is nil")
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database: ping failed: %w", err)
	}
	return nil
}
