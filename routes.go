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
	uploadsDir := http.Dir("./uploads/")
	r.ServeFiles("/public/*filepath", dir)
	r.ServeFiles("/uploads/*filepath", uploadsDir)

	dynamic := alice.New(app.sessionManager.LoadAndSave, app.authenticate)

	r.NotFound = dynamic.ThenFunc(app.notFound)

	r.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.handleHomePage))
	r.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.handleSearchPage))
	r.Handler(http.MethodGet, "/listings/:id", dynamic.ThenFunc(app.handleSingleListingPage))

	anonymous := dynamic.Append(app.requireAnonymous)

	r.Handler(http.MethodGet, "/register", anonymous.ThenFunc(app.handleRegisterPage))
	r.Handler(http.MethodPost, "/register", anonymous.ThenFunc(app.handleUserRegister))
	r.Handler(http.MethodGet, "/login", anonymous.ThenFunc(app.handleLoginPage))
	r.Handler(http.MethodPost, "/login", anonymous.ThenFunc(app.handleLogin))

	// protected routes
	protected := dynamic.Append(app.requireAuthentication)

	r.Handler(http.MethodGet, "/property", protected.ThenFunc(app.handleNewPropertyPage))
	r.Handler(http.MethodPost, "/property", protected.ThenFunc(app.handleNewProperty))
	r.Handler(http.MethodPost, "/logout", protected.ThenFunc(app.handleLogout))
	r.Handler(http.MethodGet, "/profile", protected.ThenFunc(app.handleProfilePage))

	standard := alice.New(app.requestID, app.logRequest)
	return standard.Then(r)
}
