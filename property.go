package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

func (app *application) handleHomePage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/home.html"
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, page, data)
}

const imageDir = "./uploads/"

func (app *application) handleNewProperty(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	uploadedFile, _, err := r.FormFile("thumbnail")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer uploadedFile.Close()

	b, err := io.ReadAll(uploadedFile)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	mtype := mimetype.Detect(b)
	allowedExtensions := []string{".jpg", ".jpeg", ".png"}
	ext := mtype.Extension()
	if !slices.Contains(allowedExtensions, ext) {
		app.logger.Warn("invalid image")
		app.serverError(w, r, err)
		return
	}
	name := fmt.Sprintf("banner_%s%s", uuid.NewString(), ext)
	out, err := os.Create(filepath.Join(imageDir, name))
	defer out.Close()
	_, err = out.Write(b)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	fmt.Fprintln(w, "Uploaded file")
}

func (app *application) handleNewPropertyPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create.html"
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/search.html"
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, page, data)
}
