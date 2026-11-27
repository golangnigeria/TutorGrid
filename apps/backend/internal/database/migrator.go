package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strconv"

	"github.com/tutorgrid/tutorgrid/internal/config"

	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, logger *zerolog.Logger, cfg *config.Config) error {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	// URL-encode the password
	encodedPassword := url.QueryEscape(cfg.Database.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		encodedPassword,
		hostPort,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	return MigrateConn(ctx, logger, conn)
}

// MigrateConn runs embedded tern migrations to the latest version on an existing pgx connection.
func MigrateConn(ctx context.Context, logger *zerolog.Logger, conn *pgx.Conn) error {
	return MigrateConnTo(ctx, logger, conn, -1)
}

// MigrateConnTo runs embedded tern migrations to targetVersion (-1 migrates to latest).
func MigrateConnTo(ctx context.Context, logger *zerolog.Logger, conn *pgx.Conn, targetVersion int32) error {
	m, err := tern.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("constructing database migrator: %w", err)
	}
	subtree, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("retrieving database migrations subtree: %w", err)
	}
	if err := m.LoadMigrations(subtree); err != nil {
		return fmt.Errorf("loading database migrations: %w", err)
	}
	from, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("retreiving current database migration version: %w", err)
	}

	dest := int32(len(m.Migrations))
	if targetVersion >= 0 {
		dest = targetVersion
		if err := m.MigrateTo(ctx, targetVersion); err != nil {
			return err
		}
	} else {
		if err := m.Migrate(ctx); err != nil {
			return err
		}
	}

	if from == dest {
		logger.Info().Msgf("database schema up to date, version %d", dest)
	} else {
		logger.Info().Msgf("migrated database schema, from %d to %d", from, dest)
	}
	return nil
}
