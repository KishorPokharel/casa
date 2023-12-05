package main

import (
	"html/template"
	"log"
	"net/http"
)

func handleHomePage(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		"./ui/templates/pages/home.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	err := tmpl.ExecuteTemplate(w, "layout", "Hello World")
	if err != nil {
		log.Println(err)
	}
}

func handleNewProperty(w http.ResponseWriter, r *http.Request) {
}

func handleNewPropertyPage(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		"./ui/templates/pages/property_create.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	err := tmpl.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		log.Println(err)
	}
}
