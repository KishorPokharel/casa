package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

func (app *application) handleHomePage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/home.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/register.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/login.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

const imageDir = "./"

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

	mtype, err := mimetype.DetectReader(uploadedFile)
	fmt.Println("Extension: ", mtype.Extension())
	return
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	name := uuid.NewString()
	out, err := os.Create(filepath.Join(imageDir, "banner_"+name))
	defer out.Close()
	_, err = io.Copy(out, uploadedFile)
	if err != nil {
		app.logger.Error("copy image to uploads", "msg", err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "Uploaded file")
}

func (app *application) handleNewPropertyPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create.html"
	err := app.render(w, page, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}
