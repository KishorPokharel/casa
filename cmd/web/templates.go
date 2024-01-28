package main

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/leekchan/accounting"
)

type templateData struct {
	CurrentYear     int
	Flash           string
	Form            any
	IsAuthenticated bool
	User            storage.User
	Listings        []storage.Property
	Listing         storage.Property
	SavedListings   []storage.Property
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02 Jan 2006 at 15:04")
}

var ac = accounting.Accounting{Symbol: "NPR ", Precision: 2, FormatNegative: "%s -%v"}

var functions = template.FuncMap{
	"humanDate":   humanDate,
	"formatPrice": ac.FormatMoney,
}

func (app *application) newTemplateData(r *http.Request) templateData {
	data := templateData{
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), sessionFlashKey),
		IsAuthenticated: app.isAuthenticated(r),
	}
	return data
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	files := []string{
		"./ui/templates/layout.html",
		"./ui/templates/partials/header.html",
		page,
	}

	tmpl, err := template.New(filepath.Base(page)).Funcs(functions).ParseFiles(files...)
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
