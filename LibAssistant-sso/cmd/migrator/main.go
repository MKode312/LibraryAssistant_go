package main

import (
	"LibAssistant_sso/internal/config"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	migrationsPath, ok := os.LookupEnv("MIGRATIONS_PATH")
	if !ok {
		panic("migrations-path is required")
	}

	cfg := config.MustLoad()

	time.Sleep(3 * time.Second)

	conn, err := pgx.Connect(context.Background(), cfg.Postgres.DBurl)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	m, err := migrate.New(
		"file://"+migrationsPath,
		cfg.Postgres.DBurl,
	)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")

			return
		}

		panic(err)
	}

	fmt.Println("migrations applied successfully")

}
