package main

import (
	"net/http"

	"github.com/KishorPokharel/casa/storage"
)

func (app *application) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/register.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	var u = struct {
		Username string
		Email    string
		Password string
	}{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	// TODO: validate form

	user := storage.User{
		Username: u.Username,
		Email:    u.Email,
	}
	if err := app.storage.Users.Insert(user); err != nil {

	}
}

func (app *application) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/login.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}
