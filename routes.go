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

	r.Handler(http.MethodGet, "/listings", protected.ThenFunc(app.handleNewListingPage))
	r.Handler(http.MethodPost, "/listings", protected.ThenFunc(app.handleNewListing))
	r.Handler(http.MethodGet, "/profile", protected.ThenFunc(app.handleProfilePage))
	r.Handler(http.MethodPost, "/logout", protected.ThenFunc(app.handleLogout))
	r.Handler(http.MethodPost, "/listings/:id/save", protected.ThenFunc(app.handleSaveListing))
	r.Handler(http.MethodDelete, "/listings/:id/unsave", protected.ThenFunc(app.handleUnsaveListing))

	standard := alice.New(app.methodOverride, app.requestID, app.logRequest)
	return standard.Then(r)
}
