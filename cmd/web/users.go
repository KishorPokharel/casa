package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/KishorPokharel/casa/validator"
	"github.com/julienschmidt/httprouter"
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
	data := app.newTemplateData(r)
	data.Form = userRegisterForm{}
	app.render(w, r, http.StatusOK, page, data)
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
	app.sessionManager.Put(r.Context(), sessionFlashKey, "User Registered. Please Login.")
	http.Redirect(w, r, "/users/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email    string
	Password string
	validator.Validator
}

func (app *application) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/login.html"
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, page, data)
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
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field can not be blank")

	page := "./ui/templates/pages/login.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	// Login
	id, err := app.storage.Users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, page, data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), sessionAuthKey, id)
	app.sessionManager.Put(r.Context(), sessionFlashKey, "Login Successful")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Remove(r.Context(), sessionAuthKey)
	app.sessionManager.Put(r.Context(), sessionFlashKey, "You've been logged out successfully!")

	http.Redirect(w, r, "/users/login", http.StatusSeeOther)
}

func (app *application) handleProfilePage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/profile.html"
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user

	// get my saved listings
	savedListings, err := app.storage.Property.GetSavedListings(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.SavedListings = savedListings

	// get my listings
	listings, err := app.storage.Property.GetAllForUser(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Listings = listings

	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleProfileViewPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/profile_view.html"
	userID, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}
	if app.isAuthenticated(r) {
		userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
		if userID == user.ID {
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
	}
	data := app.newTemplateData(r)
	data.User = user

	ptype := r.URL.Query().Get("propertyType")
	if ptype != "" && !slices.Contains(propertyTypes, ptype) {
		http.Redirect(w, r, fmt.Sprintf("/profile/view/%d", userID), http.StatusSeeOther)
		return
	}
	listings, err := app.storage.Property.GetAllForUser(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if ptype != "" {
		filtered := []storage.Property{}
		for _, val := range listings {
			if val.PropertyType == ptype {
				filtered = append(filtered, val)
			}
		}
		data.Listings = filtered
	} else {
		data.Listings = listings
	}

	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleProfileEditPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/profile_edit.html"
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user
	data.Form = userEditForm{
		Username: user.Username,
		Phone:    user.Phone,
	}
	app.render(w, r, http.StatusOK, page, data)
}

type userEditForm struct {
	Username string
	Phone    string
	validator.Validator
}

func (app *application) handleProfileEdit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	phone := strings.TrimSpace(r.FormValue("phone"))
	form := userEditForm{
		Username: r.FormValue("username"),
		Phone:    phone,
	}
	form.CheckField(validator.NotBlank(form.Username), "username", "This field can not be blank")
	form.CheckField(validator.MaxChars(form.Username, 20), "username", "This field can not be more than 20 chars")
	if form.Phone != "" {
		form.CheckField(len(form.Phone) == 10, "phone", "Phone must be 10 digits")
		form.CheckField(validator.Matches(form.Phone, validator.PhoneRX), "phone", "Invalid phone")
	}

	page := "./ui/templates/pages/profile_edit.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	user.Username = form.Username
	user.Phone = form.Phone
	if err := app.storage.Users.Update(user.ID, user); err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Profile Updated")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

type passwordChangeForm struct {
	CurrentPassword    string
	NewPassword        string
	ConfirmNewPassword string
	validator.Validator
}

func (app *application) handleChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/password_change.html"
	data := app.newTemplateData(r)
	data.Form = passwordChangeForm{}
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form := passwordChangeForm{
		CurrentPassword:    r.FormValue("currentPassword"),
		NewPassword:        r.FormValue("newPassword"),
		ConfirmNewPassword: r.FormValue("confirmNewPassword"),
	}

	form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.NewPassword), "newPassword", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.NewPassword, 10), "newPassword", "This field must be at least 10 characters long")
	form.CheckField(validator.NotBlank(form.ConfirmNewPassword), "confirmNewPassword", "This field cannot be blank")
	form.CheckField(form.NewPassword == form.ConfirmNewPassword, "confirmNewPassword", "Passwords do not match")

	page := "./ui/templates/pages/password_change.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	if err := app.storage.Users.PasswordUpdate(userID, form.CurrentPassword, form.NewPassword); err != nil {
		if errors.Is(err, storage.ErrInvalidCredentials) {
			form.AddFieldError("currentPassword", "Current password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, page, data)
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Password Changed")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

type forgotPasswordForm struct {
	Email string
	validator.Validator
}

func (app *application) handleForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/forgot_password.html"
	form := forgotPasswordForm{}
	data := app.newTemplateData(r)
	data.Form = form
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forgotPasswordForm{
		Email: r.FormValue("email"),
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field can not be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")

	page := "./ui/templates/pages/forgot_password.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	user, err := app.storage.Users.GetByEmail(form.Email)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			form.AddFieldError("email", "No user with provided email.")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusBadRequest, page, data)
			return
		}
		app.serverError(w, r, err)
		return
	}
	// create a password reset token for email
	token, err := app.storage.Tokens.New(user.ID, 10*time.Minute, storage.ScopePasswordReset)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	// use token and email to send mail
	app.background(func() {
		data := map[string]any{
			"passwordResetToken": token.PlainText,
			"port":               app.config.port,
		}
		err = app.mailer.Send(user.Email, "./mailer/templates/password-reset.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Check your email to reset your password to continue.")
	http.Redirect(w, r, "/users/login", http.StatusSeeOther)
}

type passwordResetForm struct {
	Token              string
	NewPassword        string
	ConfirmNewPassword string
	validator.Validator
}

func (app *application) handlePasswordResetPage(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	token := params.ByName("token")

	page := "./ui/templates/pages/password_reset.html"
	data := app.newTemplateData(r)
	data.Form = passwordResetForm{
		Token: token,
	}
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handlePasswordReset(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	token := params.ByName("token")

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := passwordResetForm{
		Token:              token,
		NewPassword:        r.FormValue("newPassword"),
		ConfirmNewPassword: r.FormValue("confirmNewPassword"),
	}

	form.CheckField(validator.NotBlank(form.NewPassword), "newPassword", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.NewPassword, 10), "newPassword", "This field must be at least 10 characters long")
	form.CheckField(validator.NotBlank(form.ConfirmNewPassword), "confirmNewPassword", "This field cannot be blank")
	form.CheckField(form.NewPassword == form.ConfirmNewPassword, "confirmNewPassword", "Passwords do not match")

	page := "./ui/templates/pages/password_reset.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	user, err := app.storage.Users.GetForToken(storage.ScopePasswordReset, form.Token)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			form.AddNonFieldError("Invalid or Expired password reset token")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, page, data)
			return
		}
		app.serverError(w, r, err)
		return
	}
	if err := app.storage.Users.PasswordReset(user.ID, form.NewPassword); err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.storage.Tokens.DeleteAllForUser(storage.ScopePasswordReset, user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Your password was reset")
	http.Redirect(w, r, "/users/login", http.StatusSeeOther)
}
