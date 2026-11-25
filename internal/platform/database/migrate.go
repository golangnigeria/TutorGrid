package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/tutorgrid/tutorgrid/migrations"
)

// MigrateUp applies all pending embedded Goose SQL migrations.
func MigrateUp(ctx context.Context, sqldb *sql.DB) error {
	provider, err := newGooseProvider(sqldb)
	if err != nil {
		return err
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("database: goose up: %w", err)
	}
	return nil
}

// MigrateDownTo rolls back embedded Goose SQL migrations to the specified version (0 = all).
func MigrateDownTo(ctx context.Context, sqldb *sql.DB, version int64) error {
	provider, err := newGooseProvider(sqldb)
	if err != nil {
		return err
	}
	if _, err := provider.DownTo(ctx, version); err != nil {
		return fmt.Errorf("database: goose down to %d: %w", version, err)
	}
	return nil
}

func newGooseProvider(sqldb *sql.DB) (*goose.Provider, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, sqldb, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("database: create goose provider: %w", err)
	}
	return provider, nil
}
