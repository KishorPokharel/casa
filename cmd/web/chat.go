package main

import (
	"fmt"
	"net/http"
)

func (app *application) handleChat(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Single Chat Page")
}

func (app *application) handleAllChatsPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "all Chat Page")
}
