package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/alexedwards/scs/v2"
)

type application struct {
	logger         *slog.Logger
	config         *config
	storage        storage.Storage
	sessionManager *scs.SessionManager
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
