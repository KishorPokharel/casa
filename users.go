package main

import (
	"errors"
	"net/http"

	"github.com/KishorPokharel/casa/storage"
	"github.com/KishorPokharel/casa/validator"
)

type userRegisterForm struct {
	Username        string
	Email           string
	Password        string
	ConfirmPassword string
	validator.Validator
}

func (app *application) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/register.html"
	app.render(w, r, http.StatusOK, page, nil)
}

func (app *application) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form := userRegisterForm{
		Username:        r.FormValue("username"),
		Email:           r.FormValue("email"),
		Password:        r.FormValue("password"),
		ConfirmPassword: r.FormValue("password2"),
	}

	// Validate Form
	form.CheckField(validator.NotBlank(form.Username), "username", "This field can not be blank")
	form.CheckField(validator.MaxChars(form.Username, 20), "username", "This field can not be more than 20 chars")

	form.CheckField(validator.NotBlank(form.Email), "email", "This field can not be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")

	form.CheckField(validator.NotBlank(form.Password), "password", "This field can not be blank")
	form.CheckField(validator.MinChars(form.Password, 10), "password", "This field can not be less than 10 chars")
	form.CheckField(form.Password == form.ConfirmPassword, "password", "Two passwords do not match")

	page := "./ui/templates/pages/register.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}
	// Register new user
	user := storage.User{
		Username: form.Username,
		Email:    form.Email,
	}
	err = user.Password.Set(form.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if err := app.storage.Users.Insert(user); err != nil {
		if errors.Is(err, storage.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email already exists")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, page, data)
			return
		}
		app.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email    string
	Password string
	validator.Validator
}

func (app *application) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/login.html"
	app.render(w, r, http.StatusOK, page, nil)
}

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form := userLoginForm{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	// Validate Form
	form.CheckField(validator.NotBlank(form.Email), "email", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field can not be blank")

	page := "./ui/templates/pages/login.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: logout
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
