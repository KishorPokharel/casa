package main

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type application struct {
	logger *slog.Logger
	config *config
}

func (app *application) routes() http.Handler {
	r := httprouter.New()

	// serve static files
	dir := http.Dir("./ui/public/")
	r.ServeFiles("/public/*filepath", dir)

	// routes
	r.HandlerFunc(http.MethodGet, "/", app.handleHomePage)

	r.HandlerFunc(http.MethodGet, "/register", app.handleRegisterPage)
	r.HandlerFunc(http.MethodGet, "/login", app.handleLoginPage)

	r.HandlerFunc(http.MethodGet, "/property", app.handleNewPropertyPage)
	r.HandlerFunc(http.MethodPost, "/property", app.handleNewProperty)

	return app.requestID(app.logRequest(r))
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

func (app *application) render(w http.ResponseWriter, page string, data any) error {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		page,
	}
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return fmt.Errorf("parse template files %v", err)
	}
	return tmpl.ExecuteTemplate(w, "layout", data)
}
