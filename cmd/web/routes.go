package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	r := httprouter.New()

	// serve static files
	r.ServeFiles("/public/*filepath", http.Dir(publicDir))
	r.ServeFiles("/uploads/*filepath", http.Dir(uploadDir))

	r.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	dynamic := alice.New(app.sessionManager.LoadAndSave, app.authenticate)

	r.NotFound = dynamic.ThenFunc(app.notFound)

	r.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.handleHomePage))
	r.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.handleSearchPage))
	r.Handler(http.MethodGet, "/listings/view/:id", dynamic.ThenFunc(app.handleSingleListingPage))
	r.Handler(http.MethodGet, "/locations", dynamic.ThenFunc(app.handleGetAllLocations))

	anonymous := dynamic.Append(app.requireAnonymous)

	r.Handler(http.MethodGet, "/users/register", anonymous.ThenFunc(app.handleRegisterPage))
	r.Handler(http.MethodPost, "/users/register", anonymous.ThenFunc(app.handleUserRegister))
	r.Handler(http.MethodGet, "/users/login", anonymous.ThenFunc(app.handleLoginPage))
	r.Handler(http.MethodPost, "/users/login", anonymous.ThenFunc(app.handleLogin))

	// protected routes
	protected := dynamic.Append(app.requireAuthentication)

	r.Handler(http.MethodPost, "/thumbnail/upload", protected.ThenFunc(app.handleFileUpload("thumbnail")))
	r.Handler(http.MethodPost, "/pictures/upload", protected.ThenFunc(app.handleFileUpload("picture")))

	// r.Handler(http.MethodGet, "/listings/create", protected.ThenFunc(app.handleNewListingPage))
	r.Handler(http.MethodGet, "/listings/create", protected.ThenFunc(app.handleNewListingPageWithFilepond))
	// r.Handler(http.MethodPost, "/listings/_create", protected.ThenFunc(app.handleNewListing))
	r.Handler(http.MethodPost, "/listings/create", protected.ThenFunc(app.handleNewListingFilepond))

	r.Handler(http.MethodGet, "/listings/saved", protected.ThenFunc(app.handleSavedListingsPage))
	r.Handler(http.MethodGet, "/listings/my", protected.ThenFunc(app.handleMyListingsPage))

	r.Handler(http.MethodGet, "/profile", protected.ThenFunc(app.handleProfilePage))
	r.Handler(http.MethodGet, "/profile/update", protected.ThenFunc(app.handleProfileEditPage))
	r.Handler(http.MethodPut, "/profile/update", protected.ThenFunc(app.handleProfileEdit))
	r.Handler(http.MethodPost, "/users/logout", protected.ThenFunc(app.handleLogout))
	r.Handler(http.MethodGet, "/change-password", protected.ThenFunc(app.handleChangePasswordPage))
	r.Handler(http.MethodPost, "/change-password", protected.ThenFunc(app.handleChangePassword))

	r.Handler(http.MethodPost, "/listings/save/:id", protected.ThenFunc(app.handleSaveListing))
	r.Handler(http.MethodDelete, "/listings/unsave/:id", protected.ThenFunc(app.handleUnsaveListing))

	standard := alice.New(app.metrics, app.methodOverride, app.requestID, app.logRequest)
	return standard.Then(r)
}
