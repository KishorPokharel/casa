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
	app.render(w, r, http.StatusOK, page, nil)
}

const imageDir = "./uploads/"

func (app *application) handleNewProperty(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		app.logger.Error("could not parse form", "msg", err.Error())
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}
	uploadedFile, _, err := r.FormFile("thumbnail")
	if err != nil {
		app.logger.Error("r.FormFile", "msg", err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	defer uploadedFile.Close()

	b, err := io.ReadAll(uploadedFile)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	mtype := mimetype.Detect(b)
	allowedExtensions := []string{".jpg", ".jpeg", ".png"}
	ext := mtype.Extension()
	if !slices.Contains(allowedExtensions, ext) {
		app.logger.Warn("invalid image")
		http.Error(w, "Invalid Image File", http.StatusBadRequest)
		return
	}
	name := fmt.Sprintf("banner_%s%s", uuid.NewString(), ext)
	out, err := os.Create(filepath.Join(imageDir, name))
	defer out.Close()
	_, err = out.Write(b)
	if err != nil {
		app.logger.Error("failed writing image", "msg", err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "Uploaded file")
}

func (app *application) handleNewPropertyPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create.html"
	app.render(w, r, http.StatusOK, page, nil)
}

func (app *application) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/search.html"
	app.render(w, r, http.StatusOK, page, nil)
}
