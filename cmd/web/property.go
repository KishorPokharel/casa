package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/KishorPokharel/casa/storage"
	"github.com/KishorPokharel/casa/validator"
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
	PropertyType string
	Title        string
	Location     string
	Price        int64
	ImageURL     string
	// FurnishStatus string
	Description string
	validator.Validator
}

// func (app *application) handleNewListing(w http.ResponseWriter, r *http.Request) {
// 	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
// 	err := r.ParseMultipartForm(10 << 20) // 10MB
// 	if err != nil {
// 		app.logger.Error(err.Error())
// 		app.clientError(w, http.StatusBadRequest)
// 		return
// 	}
// 	uploadedFile, _, err := r.FormFile("thumbnail")
// 	if err != nil {
// 		app.logger.Error(err.Error())
// 		app.clientError(w, http.StatusBadRequest)
// 		return
// 	}
// 	defer uploadedFile.Close()

// 	b, err := io.ReadAll(uploadedFile)
// 	if err != nil {
// 		app.serverError(w, r, err)
// 		return
// 	}
// 	mtype := mimetype.Detect(b)

// 	allowedExtensions := []string{".jpg", ".jpeg", ".png"}

// 	ext := mtype.Extension()
// 	if !slices.Contains(allowedExtensions, ext) {
// 		app.logger.Warn("invalid image")
// 		app.serverError(w, r, err)
// 		return
// 	}
// 	name := fmt.Sprintf("banner_%s%s", uuid.NewString(), ext)

// 	form := propertyCreateForm{
// 		Title:       r.FormValue("title"),
// 		Location:    r.FormValue("location"),
// 		ImageURL:    name,
// 		Description: r.FormValue("description"),
// 	}
// 	priceString := r.FormValue("price")

// 	// validate form
// 	form.CheckField(validator.NotBlank(priceString), "price", "This field can not be blank")
// 	form.CheckField(validator.NotBlank(form.Title), "title", "This field can not be blank")
// 	form.CheckField(validator.NotBlank(form.Location), "location", "This field can not be blank")
// 	form.CheckField(validator.NotBlank(form.Description), "description", "This field can not be blank")

// 	if strings.TrimSpace(priceString) != "" {
// 		price, err := strconv.Atoi(priceString)
// 		if err != nil {
// 			app.logger.Error(err.Error())
// 			app.clientError(w, http.StatusBadRequest)
// 			return
// 		}
// 		form.Price = int64(price)
// 		form.CheckField(form.Price > 0, "price", "Price must greater than 0")
// 	}

// 	page := "./ui/templates/pages/property_create.html"
// 	if !form.Valid() {
// 		data := app.newTemplateData(r)
// 		data.Form = form
// 		app.render(w, r, http.StatusUnprocessableEntity, page, data)
// 		return
// 	}

// 	out, err := os.Create(filepath.Join(uploadDir, name))
// 	defer out.Close()
// 	_, err = out.Write(b)
// 	if err != nil {
// 		app.serverError(w, r, err)
// 		return
// 	}
// 	property := storage.Property{
// 		Banner:      form.ImageURL,
// 		Location:    form.Location,
// 		Title:       form.Title,
// 		Description: form.Description,
// 		Price:       form.Price,
// 		UserID:      userID,
// 	}
// 	err = app.storage.Property.Insert(property)
// 	if err != nil {
// 		app.serverError(w, r, err)
// 		return
// 	}
// 	app.sessionManager.Put(r.Context(), sessionFlashKey, "Created a property")
// 	http.Redirect(w, r, "/", http.StatusSeeOther)
// }

// func (app *application) handleNewListingPage(w http.ResponseWriter, r *http.Request) {
// 	page := "./ui/templates/pages/property_create.html"
// 	data := app.newTemplateData(r)
// 	data.Form = propertyCreateForm{}
// 	app.render(w, r, http.StatusOK, page, data)
// }

func (app *application) handleNewListingPageWithFilepond(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/property_create_filepond.html"
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
	MinPrice     int64
	MaxPrice     int64
	validator.Validator
}

func (app *application) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/search.html"
	// query := strings.TrimSpace(r.URL.Query().Get("query"))
	minPriceString := strings.TrimSpace(r.URL.Query().Get("minPrice"))
	maxPriceString := strings.TrimSpace(r.URL.Query().Get("maxPrice"))

	form := queryForm{
		Location:     strings.TrimSpace(r.URL.Query().Get("query")),
		PropertyType: strings.TrimSpace(r.URL.Query().Get("propertyType")),
	}

	permitted := slices.Concat([]string{"", "any"}, propertyTypes)
	form.CheckField(validator.PermittedValue(form.PropertyType, permitted...), "propertyType", "Invalid property type")
	if minPriceString != "" {
		minPrice, err := strconv.Atoi(minPriceString)
		if err != nil {
			form.CheckField(false, "minPrice", "Invalid input")
		} else {
			form.MinPrice = int64(minPrice)
			form.CheckField(form.MinPrice >= 0, "minPrice", "Value must be greater or equal to 0")
		}
	}
	if maxPriceString != "" {
		maxPrice, err := strconv.Atoi(maxPriceString)
		if err != nil {
			form.CheckField(false, "maxPrice", "Invalid input")
		} else {
			form.MaxPrice = int64(maxPrice)
			form.CheckField(form.MaxPrice >= 0, "maxPrice", "Value must be greater than 0")
		}
	}
	if minPriceString != "" && maxPriceString != "" {
		if !(form.MinPrice == 0 && form.MaxPrice == 0) {
			form.CheckField(form.MinPrice < form.MaxPrice, "maxPrice", "Min Price should be smaller than Max Price")
		}
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
		MinPrice:     form.MinPrice,
		MaxPrice:     form.MaxPrice,
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

	// get my saved listings
	savedListings, err := app.storage.Property.GetSavedListings(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.SavedListings = savedListings

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

	// get my listings
	listings, err := app.storage.Property.GetAllForUser(userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Listings = listings

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
