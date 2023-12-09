package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	r := httprouter.New()

	// serve static files
	dir := http.Dir("./ui/public/")
	r.ServeFiles("/public/*filepath", dir)

	// routes
	r.HandlerFunc(http.MethodGet, "/", app.handleHomePage)

	r.HandlerFunc(http.MethodGet, "/register", app.handleRegisterPage)
	r.HandlerFunc(http.MethodPost, "/register", app.handleUserRegister)
	r.HandlerFunc(http.MethodGet, "/login", app.handleLoginPage)

	r.HandlerFunc(http.MethodGet, "/property", app.handleNewPropertyPage)
	r.HandlerFunc(http.MethodPost, "/property", app.handleNewProperty)

	return app.requestID(app.logRequest(r))
}
