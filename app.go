package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type application struct {
	logger *slog.Logger
	config *config
}

func (app *application) routes() *httprouter.Router {
	r := httprouter.New()

	// serve static files
	dir := http.Dir("./ui/public/")
	r.ServeFiles("/public/*filepath", dir)

	// routes
	r.HandlerFunc(http.MethodGet, "/", handleHomePage)
	r.HandlerFunc(http.MethodGet, "/property", handleNewPropertyPage)
	r.HandlerFunc(http.MethodPost, "/property", handleNewProperty)

	return r
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
