package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
)

type config struct {
	port int
}

func main() {
	dsn := os.Getenv("CASA_DB_DSN")
	config := &config{
		port: 3000,
	}
	app := &application{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		config: config,
	}
	db, err := openDB(dsn)
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}

	// storage
	app.storage = storage.New(db)

	// session manager
	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	app.sessionManager = sessionManager

	if err := app.run(); err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
