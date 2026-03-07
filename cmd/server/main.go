package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/config"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/db"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/web"
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

	supplierStore := suppliers.NewPostgresStore(sqlDB)
	bookStore := books.NewPostgresStore(sqlDB)
	bundleStore := bundles.NewPostgresStore(sqlDB)
	server := web.NewServerWithStores(supplierStore, bookStore, bundleStore)

	addr := ":8080"
	log.Printf("SBD server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
