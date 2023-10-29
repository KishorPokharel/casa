package main

import (
	"html/template"
	"net/http"

	"github.com/KishorPokharel/casa/ui"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(ui.Files, "templates/home.html"))
	tmpl.Execute(w, "Hello World")
}
