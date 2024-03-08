package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/julienschmidt/httprouter"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func (app *application) handleChatPage(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	roomID := params.ByName("id")
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

	ok, err := app.storage.Messages.CanAccessRoom(userID, roomID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if !ok {
		app.clientError(w, http.StatusForbidden)
		return
	}

	messages, err := app.storage.Messages.GetAllMessages(roomID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	chatOtherUser, err := app.storage.Messages.GetOtherUserOfRoom(userID, roomID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	page := "./ui/templates/pages/single-chat.html"
	data := app.newTemplateData(r)
	data.Messages = messages
	data.ChatOtherUser = chatOtherUser
	data.AuthenticatedUser = user
	data.RoomID = roomID
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

func (app *application) handleWSChat(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	roomID := params.ByName("id")
	userID := app.sessionManager.GetInt64(r.Context(), sessionAuthKey)

	// check if user with userID can access room with roomID
	ok, err := app.storage.Messages.CanAccessRoom(userID, roomID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if !ok {
		app.clientError(w, http.StatusForbidden)
		return
	}

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		app.logger.Error("could not accept ws connection", err)
		app.serverError(w, r, err)
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	var v interface{}
	err = wsjson.Read(ctx, c, &v)
	if err != nil {
		// ...
	}

	log.Printf("received: %v", v)

	c.Close(websocket.StatusNormalClosure, "")
}
