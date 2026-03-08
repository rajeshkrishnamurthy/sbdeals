package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/config"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/db"
)

func main() {
	ctx := context.Background()

	databaseURL, err := config.DatabaseURLFromEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	migrator := db.NewMigrator(db.EmbeddedMigrations, db.EmbeddedMigrationsDir)
	if err := migrator.Up(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
}
