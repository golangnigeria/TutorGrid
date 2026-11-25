package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/tutorgrid/tutorgrid/internal/platform/database"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	dsn := os.Getenv("DATABASE_MIGRATION_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_MIGRATION_URL is required")
		os.Exit(1)
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	defer sqldb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch *direction {
	case "up":
		if err := database.MigrateUp(ctx, sqldb); err != nil {
			fmt.Fprintf(os.Stderr, "migrate up failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations applied successfully")
	case "down":
		if err := database.MigrateDownTo(ctx, sqldb, 0); err != nil {
			fmt.Fprintf(os.Stderr, "migrate down failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations rolled back successfully")
	default:
		fmt.Fprintf(os.Stderr, "unknown direction %q (expected up or down)\n", *direction)
		os.Exit(1)
	}
}
