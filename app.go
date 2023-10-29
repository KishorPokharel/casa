package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KishorPokharel/casa/ui"
)

type application struct {
	logger *slog.Logger
	config *config
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// serve static files
	fs := http.FileServer(http.FS(ui.Files))
	mux.Handle("/public/", fs)

	mux.HandleFunc("/", handleHome)

	return mux
}

func (app *application) run() error {
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	app.logger.Info("server started", "port", app.config.port)
	return srv.ListenAndServe()
}
