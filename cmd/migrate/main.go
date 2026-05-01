package main

import (
	"database/sql"
	"errors"
	"expire-share/internal/config"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: migrate [up/down]")
	}

	cfg := config.MustLoad()

	cmd := os.Args[1]

	db, err := sql.Open("mysql", cfg.DbConnectionString)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			log.Fatalf("failed to close database: %v", err)
		}
	}(db)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	mgr, err := migrate.New("file://migrations", "mysql://"+cfg.DbConnectionString)
	if err != nil {
		log.Fatalf("failed to init migrator: %v", err)
	}

	operations := map[string]func() error{
		"up":   mgr.Up,
		"down": func() error { return mgr.Steps(-1) },
	}

	fn, ok := operations[cmd]
	if !ok {
		log.Fatalf("unknown command: %s", cmd)
	}

	if err := fn(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to migrate table: %v", err)
	}

	fmt.Println("migration complete")
}
