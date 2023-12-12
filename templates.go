package main

import (
	"fmt"
	"html/template"
	"net/http"
)

type templateData struct {
	Form any
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{}
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data any) error {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		page,
	}
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return fmt.Errorf("parse template files %v", err)
	}
	return tmpl.ExecuteTemplate(w, "layout", data)
}
