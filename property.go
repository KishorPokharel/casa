package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/KishorPokharel/casa/storage"
	"github.com/KishorPokharel/casa/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

func (app *application) handleHomePage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/home.html"
	data := app.newTemplateData(r)
	listings, err := app.storage.Property.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Listings = listings
	app.render(w, r, http.StatusOK, page, data)
}

const imageDir = "./uploads/"

type propertyCreateForm struct {
	// ListingType   string
	// PropertyType  string
	Title string
	// Province      string
	Location string
	Price    int64
	ImageURL string
	// FurnishStatus string
	Description string
	validator.Validator
}

func (app *application) handleNewProperty(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}
	uploadedFile, _, err := r.FormFile("thumbnail")
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
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

	form := propertyCreateForm{
		Title:       r.FormValue("title"),
		Location:    r.FormValue("location"),
		ImageURL:    name,
		Description: r.FormValue("description"),
	}
	priceString := r.FormValue("price")

	// validate form
	form.CheckField(validator.NotBlank(priceString), "price", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Title), "title", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Location), "location", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Description), "description", "This field can not be blank")

	if strings.TrimSpace(priceString) != "" {
		price, err := strconv.Atoi(priceString)
		if err != nil {
			app.logger.Error(err.Error())
			app.clientError(w, http.StatusBadRequest)
			return
		}
		form.Price = int64(price)
		form.CheckField(form.Price > 0, "price", "Price must greater than 0")
	}

	page := "./ui/templates/pages/property_create.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	out, err := os.Create(filepath.Join(imageDir, name))
	defer out.Close()
	_, err = out.Write(b)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	property := storage.Property{
		Banner:      form.ImageURL,
		Location:    form.Location,
		Title:       form.Title,
		Description: form.Description,
		Price:       form.Price,
	}
	err = app.storage.Property.Insert(property)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), sessionFlashKey, "Created a property")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) handleNewPropertyPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create.html"
	data := app.newTemplateData(r)
	data.Form = propertyCreateForm{}
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleSingleListingPage(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}
	p, err := app.storage.Property.Get(id)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}
	page := "./ui/templates/pages/property_single.html"
	data := app.newTemplateData(r)
	data.Listing = p
	app.render(w, r, http.StatusOK, page, data)
}

type queryForm struct {
	Query string
}

func (app *application) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/search.html"
	data := app.newTemplateData(r)
	query := r.URL.Query().Get("query")
	listings, err := app.storage.Property.Search(query)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Listings = listings
	data.Form = queryForm{
		Query: query,
	}
	app.render(w, r, http.StatusOK, page, data)
}
