package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	r := httprouter.New()

	// serve static files
	r.ServeFiles("/public/*filepath", http.Dir(publicDir))
	r.ServeFiles("/uploads/*filepath", http.Dir(uploadDir))

	dynamic := alice.New(app.sessionManager.LoadAndSave, app.authenticate)

	r.NotFound = dynamic.ThenFunc(app.notFound)

	r.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.handleHomePage))
	r.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.handleSearchPage))
	r.Handler(http.MethodGet, "/listings/view/:id", dynamic.ThenFunc(app.handleSingleListingPage))

	anonymous := dynamic.Append(app.requireAnonymous)

	r.Handler(http.MethodGet, "/users/register", anonymous.ThenFunc(app.handleRegisterPage))
	r.Handler(http.MethodPost, "/users/register", anonymous.ThenFunc(app.handleUserRegister))
	r.Handler(http.MethodGet, "/users/login", anonymous.ThenFunc(app.handleLoginPage))
	r.Handler(http.MethodPost, "/users/login", anonymous.ThenFunc(app.handleLogin))

	// protected routes
	protected := dynamic.Append(app.requireAuthentication)

	r.Handler(http.MethodGet, "/listings", protected.ThenFunc(app.handleNewListingPage))
	r.Handler(http.MethodPost, "/listings", protected.ThenFunc(app.handleNewListing))
	r.Handler(http.MethodGet, "/profile", protected.ThenFunc(app.handleProfilePage))
	r.Handler(http.MethodPost, "/users/logout", protected.ThenFunc(app.handleLogout))
	r.Handler(http.MethodPost, "/listings/:id/save", protected.ThenFunc(app.handleSaveListing))
	r.Handler(http.MethodDelete, "/listings/:id/unsave", protected.ThenFunc(app.handleUnsaveListing))

	standard := alice.New(app.methodOverride, app.requestID, app.logRequest)
	return standard.Then(r)
}
