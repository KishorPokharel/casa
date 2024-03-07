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

func (app *application) handleFileUpload(name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(5 << 20) // 5MB
		if err != nil {
			app.logger.Error(err.Error())
			app.clientError(w, http.StatusBadRequest)
			return
		}
		uploadedFile, _, err := r.FormFile(name)
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

		name := fmt.Sprintf("%s%s", uuid.NewString(), ext)
		out, err := os.Create(filepath.Join(tmpDir, name))
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		defer out.Close()

		_, err = out.Write(b)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		fmt.Fprint(w, name)
	}
}

type propertyCreateForm2 struct {
	Title        string
	PropertyType string
	Location     string
	Price        int64
	Latitude     float64
	Longitude    float64
	Thumbnail    string
	Pictures     []string
	Description  string
	validator.Validator
}

var propertyTypes = []string{"land", "house"}

func (app *application) handleNewListingFilepond(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)

	err := r.ParseMultipartForm(5 << 20) // 5MB
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := propertyCreateForm2{
		Title:        r.FormValue("title"),
		PropertyType: r.FormValue("propertyType"),
		Location:     r.FormValue("location"),
		Thumbnail:    r.FormValue("thumbnail"),
		Description:  r.FormValue("description"),
		Pictures:     r.Form["picture"],
	}
	priceString := r.FormValue("price")
	latitudeString := r.FormValue("latitude")
	longitudeString := r.FormValue("longitude")

	// validate form
	form.CheckField(validator.NotBlank(priceString), "price", "This field can not be blank")
	form.CheckField(validator.NotBlank(latitudeString), "latitude", "This field can not be blank")
	form.CheckField(validator.NotBlank(longitudeString), "longitude", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Title), "title", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Location), "location", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Description), "description", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Thumbnail), "thumbnail", "This field can not be blank")
	form.CheckField(validator.PermittedValue(form.PropertyType, propertyTypes...), "propertyType", "Invalid property type")

	// validate pictures array
	for _, picture := range form.Pictures {
		form.CheckField(validator.NotBlank(picture), "picture", "Invalid picture data")
	}

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
	if strings.TrimSpace(latitudeString) != "" {
		latitude, err := strconv.ParseFloat(latitudeString, 64)
		if err != nil {
			form.CheckField(false, "latitude", "Invalid input")
		} else {
			// TODO: maybe check whats valid latitude/longitude
			form.Latitude = latitude
		}
	}
	if strings.TrimSpace(longitudeString) != "" {
		longitude, err := strconv.ParseFloat(longitudeString, 64)
		if err != nil {
			form.CheckField(false, "longitude", "Invalid input")
		} else {
			// TODO: maybe check whats valid latitude/longitude
			form.Longitude = longitude
		}
	}

	page := "./ui/templates/pages/property_create_filepond.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	// move files from tmpDir to uploadDir
	// move thumbnail image
	thumbnail := filepath.Base(form.Thumbnail)
	oldImagePath := filepath.Join(tmpDir, thumbnail)
	newImagePath := filepath.Join(uploadDir, thumbnail)
	if err := os.Rename(oldImagePath, newImagePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		app.serverError(w, r, err)
		return
	}

	// move additional pictures
	for _, picture := range form.Pictures {
		picture = filepath.Base(picture)
		oldImagePath := filepath.Join(tmpDir, picture)
		newImagePath := filepath.Join(uploadDir, picture)
		if err := os.Rename(oldImagePath, newImagePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				app.clientError(w, http.StatusBadRequest)
				return
			}
			app.serverError(w, r, err)
			return
		}
	}

	property := storage.Property{
		Banner:       form.Thumbnail,
		Location:     form.Location,
		Title:        form.Title,
		Description:  form.Description,
		PropertyType: form.PropertyType,
		Latitude:     form.Latitude,
		Longitude:    form.Longitude,
		Price:        form.Price,
		UserID:       userID,
		Pictures:     form.Pictures,
	}

	err = app.storage.Property.Insert(property)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Created a property")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
