package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/KishorPokharel/casa/storage"
	"github.com/julienschmidt/httprouter"

	"github.com/gorilla/websocket"
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

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

type message struct {
	SenderID  int64  `json:"sender_id,omitempty"`
	Content   string `json:"content,omitempty"`
	RoomID    string
	CreatedAt time.Time `json:"created_at"`
}

type client struct {
	socket *websocket.Conn
	user   storage.User
	app    *application
	roomID string
	hub    *hub
	send   chan message
}

func (c *client) read() {
	defer c.socket.Close()
	for {
		var m message
		err := c.socket.ReadJSON(&m)
		if err != nil {
			c.app.logger.Error("[Client] could not read message", "err", err)
			return
		}
		fmt.Println("Client sent message", m)
		msg, err := c.app.storage.Messages.Insert(m.Content, c.roomID, c.user.ID)
		if err != nil {
			errmsg := "message insert failed"
			c.app.logger.Error(errmsg, "err", err)
			return
		}
		m.RoomID = c.roomID
		m.CreatedAt = msg.CreatedAt
		m.SenderID = c.user.ID
		c.hub.forward <- m
	}
}

func (c *client) write() {
	defer c.socket.Close()
	for {
		for m := range c.send {
			if err := c.socket.WriteJSON(m); err != nil {
				c.app.logger.Error("[Client] could not write message", "err", err)
				return
			}
		}
	}
}

type hub struct {
	forward chan message
	join    chan *client
	leave   chan *client
	clients map[*client]bool
}

func newHub() *hub {
	return &hub{
		forward: make(chan message),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.join:
			h.clients[client] = true
		case m := <-h.forward:
			for client := range h.clients {
				if client.roomID == m.RoomID {
					client.send <- m
				}
			}
		case client := <-h.leave:
			delete(h.clients, client)
			close(client.send)
		}
	}
}

func (app *application) handleWSChat(w http.ResponseWriter, r *http.Request) {
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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.logger.Error("Could not upgrade connection: ", err)
		return
	}

	client := &client{
		socket: conn,
		user:   user,
		roomID: roomID,
		send:   make(chan message),
		hub:    app.hub,
		app:    app,
	}

	client.hub.join <- client
	defer func() {
		client.hub.leave <- client
	}()
	go client.read()
	client.write()
}
