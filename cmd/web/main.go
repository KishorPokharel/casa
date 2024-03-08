package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
)

type config struct {
	port  int
	debug bool
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode")
	port := flag.Int("port", 3000, "Port to listen")
	dsn := flag.String("db-dsn", os.Getenv("CASA_DB_DSN"), "Db dsn")

	flag.Parse()

	config := &config{
		port:  *port,
		debug: *debug,
	}
	app := &application{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		config: config,
	}
	db, err := openDB(*dsn)
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

	// create upload dir and tmp dir if not exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, os.ModePerm)
	}
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.Mkdir(tmpDir, os.ModePerm)
	}

	app.hub = newHub()
	go app.hub.run()

	// publish stats
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

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
