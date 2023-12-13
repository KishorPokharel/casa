package main

import (
	"bytes"
	"html/template"
	"net/http"
)

type templateData struct {
	Flash           string
	Form            any
	IsAuthenticated bool
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
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
