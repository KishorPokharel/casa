package main

import (
	"bytes"
	"html/template"
	"net/http"
)

type templateData struct {
	Form any
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{}
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
