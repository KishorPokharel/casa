package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/KishorPokharel/casa/storage"
	"github.com/KishorPokharel/casa/validator"
	"github.com/julienschmidt/httprouter"
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

type propertyCreateForm struct {
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

func (app *application) handleNewListingPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create_filepond.html"
	data := app.newTemplateData(r)
	data.Form = propertyCreateForm{}
	app.render(w, r, http.StatusOK, page, data)
}

var propertyTypes = []string{"land", "house"}

func (app *application) handleNewListing(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)

	err := r.ParseMultipartForm(5 << 20) // 5MB
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := propertyCreateForm{
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

func (app *application) handleSingleListingPage(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}
	p, err := app.storage.Property.GetByID(id)
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
	if app.isAuthenticated(r) {
		userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
		user, err := app.storage.Users.Get(userID)
		if err != nil {
			if errors.Is(err, storage.ErrNoRecord) {
				http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			} else {
				app.serverError(w, r, err)
			}
			return
		}
		data.User = user
		// check if the post is saved
		// if yes set Listing.Saved to true
		saved, err := app.storage.Property.IsSaved(user.ID, p.ID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		if saved {
			data.Listing.Saved = true
		}
	}
	app.render(w, r, http.StatusOK, page, data)
}

type queryForm struct {
	Location     string
	PropertyType string
	MinPrice     string
	MaxPrice     string
	validator.Validator
}

func (app *application) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/search.html"

	form := queryForm{
		Location:     strings.TrimSpace(r.URL.Query().Get("query")),
		PropertyType: strings.TrimSpace(r.URL.Query().Get("propertyType")),
		MinPrice:     strings.TrimSpace(r.URL.Query().Get("minPrice")),
		MaxPrice:     strings.TrimSpace(r.URL.Query().Get("maxPrice")),
	}

	permitted := slices.Concat([]string{"", "any"}, propertyTypes)
	form.CheckField(validator.PermittedValue(form.PropertyType, permitted...), "propertyType", "Invalid property type")

	var minPrice, maxPrice int64
	if form.MinPrice != "" {
		minPriceInt, err := strconv.Atoi(form.MinPrice)
		if err != nil {
			form.CheckField(false, "minPrice", "Invalid input")
		} else {
			minPrice = int64(minPriceInt)
			form.CheckField(minPrice >= 0, "minPrice", "Value must be greater or equal to 0")
		}
	}
	if form.MaxPrice != "" {
		maxPriceInt, err := strconv.Atoi(form.MaxPrice)
		if err != nil {
			form.CheckField(false, "maxPrice", "Invalid input")
		} else {
			maxPrice = int64(maxPriceInt)
			form.CheckField(maxPrice > 0, "maxPrice", "Value must be greater than 0")
		}
	}
	if form.MinPrice != "" && form.MaxPrice != "" {
		form.CheckField(minPrice < maxPrice, "maxPrice", "Min Price should be smaller than Max Price")
	}
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	filter := storage.PropertyFilter{
		Location:     form.Location,
		PropertyType: form.PropertyType,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
	}
	min, max, err := app.storage.Property.GetMinMaxPrice()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if form.MinPrice == "" {
		filter.MinPrice = min
	}
	if form.MaxPrice == "" {
		filter.MaxPrice = max
	}
	listings, err := app.storage.Property.Search2(filter)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Form = form
	data.Listings = listings
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleSaveListing(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}

	exists, err := app.storage.Property.ExistsWithID(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if !exists {
		app.notFound(w, r)
		return
	}

	// TODO: clean this
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	redirectUrl := fmt.Sprintf("/listings/view/%d", id)
	err = app.storage.Property.Save(user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateSave) {
			app.sessionManager.Put(r.Context(), sessionFlashKey, "Listing Already Saved")
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), sessionFlashKey, "Listing Saved")
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

func (app *application) handleUnsaveListing(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}

	exists, err := app.storage.Property.ExistsWithID(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if !exists {
		app.notFound(w, r)
		return
	}

	// TODO: clean this
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.storage.Property.Unsave(user.ID, id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	redirectUrl := fmt.Sprintf("/listings/view/%d", id)
	app.sessionManager.Put(r.Context(), sessionFlashKey, "Listing Unsaved")
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

func (app *application) handleSavedListingsPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/my_saved.html"
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user

	ptype := r.URL.Query().Get("propertyType")
	if ptype != "" && !slices.Contains(propertyTypes, ptype) {
		http.Redirect(w, r, "/listings/saved", http.StatusSeeOther)
		return
	}

	// get my saved listings
	savedListings, err := app.storage.Property.GetSavedListings(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if ptype != "" {
		filtered := []storage.Property{}
		for _, val := range savedListings {
			if val.PropertyType == ptype {
				filtered = append(filtered, val)
			}
		}
		data.SavedListings = filtered
	} else {
		data.SavedListings = savedListings
	}

	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleMyListingsPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/my_listings.html"
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user

	ptype := r.URL.Query().Get("propertyType")
	if ptype != "" && !slices.Contains(propertyTypes, ptype) {
		http.Redirect(w, r, "/listings/my", http.StatusSeeOther)
		return
	}

	// get my listings
	listings, err := app.storage.Property.GetAllForUser(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if ptype != "" {
		filtered := []storage.Property{}
		for _, val := range listings {
			if val.PropertyType == ptype {
				filtered = append(filtered, val)
			}
		}
		data.Listings = filtered
	} else {
		data.Listings = listings
	}

	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleGetAllLocations(w http.ResponseWriter, r *http.Request) {
	locationQuery := strings.TrimSpace(r.URL.Query().Get("query"))
	locations, err := app.storage.Property.GetAllLocations()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	matches := RankFind(locationQuery, locations)
	sort.Sort(matches)
	data := map[string]any{
		"matches": matches,
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

type editListingForm struct {
	Title        string
	Location     string
	Latitude     float64
	Longitude    float64
	PropertyType string
	Price        int64
	Available    bool
	Thumbnail    string
	Description  string
	Pictures     []string
	validator.Validator
}

func (app *application) handleEditListingPage(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}

	property, err := app.storage.Property.GetByID(id)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	if property.UserID != userID {
		app.clientError(w, http.StatusForbidden)
		return
	}
	page := "./ui/templates/pages/property_edit.html"
	data := app.newTemplateData(r)
	data.Listing = property
	data.Form = editListingForm{
		Title:        property.Title,
		Location:     property.Location,
		Latitude:     property.Latitude,
		Longitude:    property.Longitude,
		PropertyType: property.PropertyType,
		Price:        property.Price,
		Available:    property.Available,
		Thumbnail:    "",
		Description:  property.Description,
		Pictures:     []string{},
	}
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleEditListing(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}

	property, err := app.storage.Property.GetByID(id)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	if property.UserID != userID {
		app.clientError(w, http.StatusForbidden)
		return
	}

	form := editListingForm{
		Title:        r.FormValue("title"),
		PropertyType: r.FormValue("propertyType"),
		Location:     r.FormValue("location"),
		Thumbnail:    strings.TrimSpace(r.FormValue("thumbnail")),
		Description:  r.FormValue("description"),
	}
	pictures := r.Form["picture"]

	priceString := r.FormValue("price")
	latitudeString := r.FormValue("latitude")
	longitudeString := r.FormValue("longitude")

	availableString := r.FormValue("available")
	form.Available = availableString != ""

	// validate form
	form.CheckField(validator.NotBlank(priceString), "price", "This field can not be blank")
	form.CheckField(validator.NotBlank(latitudeString), "latitude", "This field can not be blank")
	form.CheckField(validator.NotBlank(longitudeString), "longitude", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Title), "title", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Location), "location", "This field can not be blank")
	form.CheckField(validator.NotBlank(form.Description), "description", "This field can not be blank")
	form.CheckField(validator.PermittedValue(form.PropertyType, propertyTypes...), "propertyType", "Invalid property type")

	for _, picture := range pictures {
		if strings.TrimSpace(picture) != "" {
			form.Pictures = append(form.Pictures, picture)
		}
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

	page := "./ui/templates/pages/property_edit.html"
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.Listing = property
		app.render(w, r, http.StatusUnprocessableEntity, page, data)
		return
	}

	// move files from tmpDir to uploadDir
	// move thumbnail image
	if form.Thumbnail != "" {
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
	property.Title = form.Title
	property.Location = form.Location
	property.Latitude = form.Latitude
	property.Longitude = form.Longitude
	property.PropertyType = form.PropertyType
	property.Price = form.Price
	property.Available = form.Available
	property.Description = form.Description
	if form.Thumbnail != "" {
		property.Banner = form.Thumbnail
	}
	property.Pictures = form.Pictures

	if err := app.storage.Property.Update(property); err != nil {
		app.serverError(w, r, err)
		return
	}

	redirectURL := fmt.Sprintf("/listings/view/%d", property.ID)
	app.sessionManager.Put(r.Context(), sessionFlashKey, "Updated property")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *application) handleDeletePicture(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}
	params := httprouter.ParamsFromContext(r.Context())
	pictureID := params.ByName("picture")

	property, err := app.storage.Property.GetByID(propertyID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}
	if property.UserID != userID {
		app.clientError(w, http.StatusForbidden)
		return
	}
	if !slices.Contains(property.Pictures, pictureID) {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if err := app.storage.Property.DeletePicture(propertyID, pictureID); err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), sessionFlashKey, "Image Deleted")
	redirectURL := fmt.Sprintf("/listings/view/%d", propertyID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
