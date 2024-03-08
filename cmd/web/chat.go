package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/KishorPokharel/casa/storage"
)

func (app *application) handleChat(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/single-chat.html"
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleAllChatsPage(w http.ResponseWriter, r *http.Request) {
	page := "./ui/templates/pages/chat-all.html"
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	user, err := app.storage.Users.Get(userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.clientError(w, http.StatusBadRequest)
			return
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	rooms, err := app.storage.Messages.GetAllRoomsForUser(user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data := app.newTemplateData(r)
	data.Rooms = rooms
	app.render(w, r, http.StatusOK, page, data)
}

func (app *application) handleMessageOwner(w http.ResponseWriter, r *http.Request) {
	ownerID, err := app.readIDParam(r)
	if err != nil {
		app.notFound(w, r)
		return
	}
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)
	_, err = app.storage.Users.Get(ownerID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.clientError(w, http.StatusBadRequest)
			return
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	roomID, err := app.storage.Messages.CheckRoomExists(userID, ownerID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecord) {
			app.logger.Info("room not exists, creating new one")
			roomID, err := app.storage.Messages.NewRoom(userID, ownerID)
			if err != nil {
			}
			redirectURL := fmt.Sprintf("/chat/%s", roomID)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	redirectURL := fmt.Sprintf("/chat/%s", roomID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
