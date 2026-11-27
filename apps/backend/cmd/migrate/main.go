package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/database"
	"github.com/tutorgrid/tutorgrid/internal/logger"
)

func main() {
	targetVersion := flag.Int("to", -1, "target migration version (-1 migrates to latest)")
	flag.Parse()

	log := zerolog.New(logger.NewConsoleWriter(os.Stderr)).With().Timestamp().Str("component", "migrator").Logger()

	dsn := os.Getenv("TUTORGRID_DATABASE_MIGRATION_URL")
	if dsn == "" {
		log.Fatal().Msg("TUTORGRID_DATABASE_MIGRATION_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database for migrations")
	}
	defer conn.Close(ctx)

	if err := database.MigrateConnTo(ctx, &log, conn, int32(*targetVersion)); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
}
