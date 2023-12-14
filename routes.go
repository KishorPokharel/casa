package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	r := httprouter.New()

	r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	// serve static files
	dir := http.Dir("./ui/public/")
	r.ServeFiles("/public/*filepath", dir)

	dynamic := alice.New(app.sessionManager.LoadAndSave, app.authenticate)

	r.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.handleHomePage))
	r.Handler(http.MethodGet, "/register", dynamic.ThenFunc(app.handleRegisterPage))
	r.Handler(http.MethodPost, "/register", dynamic.ThenFunc(app.handleUserRegister))
	r.Handler(http.MethodGet, "/login", dynamic.ThenFunc(app.handleLoginPage))
	r.Handler(http.MethodPost, "/login", dynamic.ThenFunc(app.handleLogin))
	r.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.handleSearchPage))

	// protected routes
	protected := dynamic.Append(app.requireAuthentication)

	r.Handler(http.MethodGet, "/property", protected.ThenFunc(app.handleNewPropertyPage))
	r.Handler(http.MethodPost, "/property", protected.ThenFunc(app.handleNewProperty))
	r.Handler(http.MethodPost, "/logout", protected.ThenFunc(app.handleLogout))
	r.Handler(http.MethodGet, "/profile", protected.ThenFunc(app.handleProfilePage))

	standard := alice.New(app.requestID, app.logRequest)
	return standard.Then(r)
}
