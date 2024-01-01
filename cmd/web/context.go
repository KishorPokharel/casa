package main

import (
	"context"
	"net/http"
)

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")

const requestIDContextKey = contextKey("request_id")

func contextSetRequestID(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), requestIDContextKey, id)
	return r.WithContext(ctx)
}

func contextGetRequestID(r *http.Request) string {
	requestID, ok := r.Context().Value(requestIDContextKey).(string)
	if !ok {
		panic("missing request ID in request context")
	}
	return requestID
}
