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

	"github.com/KishorPokharel/casa/mailer"
	"github.com/KishorPokharel/casa/storage"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
)

type smtp struct {
	host     string
	port     int
	username string
	password string
	sender   string
}

type config struct {
	port  int
	debug bool
	smtp  smtp
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode")
	port := flag.Int("port", 3000, "Port to listen")
	dsn := flag.String("db-dsn", os.Getenv("CASA_DB_DSN"), "Db dsn")

	smtpHost := flag.String("smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	smtpPort := flag.Int("smtp-port", 2525, "SMTP port")
	smtpUsername := flag.String("smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	smtpPassword := flag.String("smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	smtpSender := flag.String("smtp-sender", "Casa <no-reply@casa.inc>", "SMTP sender")

	flag.Parse()

	config := &config{
		port:  *port,
		debug: *debug,
		smtp: smtp{
			host:     *smtpHost,
			port:     *smtpPort,
			username: *smtpUsername,
			password: *smtpPassword,
			sender:   *smtpSender,
		},
	}

	app := &application{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		config: config,
		mailer: mailer.New(
			config.smtp.host,
			config.smtp.port,
			config.smtp.username,
			config.smtp.password,
			config.smtp.sender,
		),
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
