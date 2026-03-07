package db

import "embed"

const EmbeddedMigrationsDir = "migrations"

//go:embed migrations/*.sql
var EmbeddedMigrations embed.FS
