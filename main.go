package main

import (
	"log/slog"
	"os"
)

type config struct {
	port int
}

func main() {
	config := &config{
		port: 3000,
	}
	app := &application{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		config: config,
	}
	if err := app.run(); err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}
