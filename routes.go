package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	r := httprouter.New()

	// serve static files
	dir := http.Dir("./ui/public/")
	r.ServeFiles("/public/*filepath", dir)

	dynamic := alice.New(app.sessionManager.LoadAndSave)
	// routes
	r.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.handleHomePage))

	r.Handler(http.MethodGet, "/register", dynamic.ThenFunc(app.handleRegisterPage))
	r.Handler(http.MethodPost, "/register", dynamic.ThenFunc(app.handleUserRegister))
	r.Handler(http.MethodGet, "/login", dynamic.ThenFunc(app.handleLoginPage))
	r.Handler(http.MethodPost, "/login", dynamic.ThenFunc(app.handleLogin))
	r.Handler(http.MethodPost, "/logout", dynamic.ThenFunc(app.handleLogout))

	r.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.handleSearchPage))
	r.Handler(http.MethodGet, "/property", dynamic.ThenFunc(app.handleNewPropertyPage))
	r.Handler(http.MethodPost, "/property", dynamic.ThenFunc(app.handleNewProperty))

	standard := alice.New(app.requestID, app.logRequest)
	return standard.Then(r)
}
