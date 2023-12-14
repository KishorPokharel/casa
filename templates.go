package main

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/KishorPokharel/casa/storage"
)

type templateData struct {
	Flash           string
	Form            any
	IsAuthenticated bool
	User            storage.User
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{
		Flash:           app.sessionManager.PopString(r.Context(), sessionFlashKey),
		IsAuthenticated: app.isAuthenticated(r),
	}
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		page,
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "layout", data); err != nil {
		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(status)
	buf.WriteTo(w)
}
