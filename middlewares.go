package main

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// request id
type requestIDKey string

const requestIDContextKey = requestIDKey("request_id")

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

func (app *application) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		r = contextSetRequestID(r, id)
		next.ServeHTTP(w, r)
	})
}

// Logger
type StatusResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (srw *StatusResponseWriter) WriteHeader(statusCode int) {
	srw.Status = statusCode
	srw.ResponseWriter.WriteHeader(statusCode)
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srw := &StatusResponseWriter{
			ResponseWriter: w,
			Status:         200,
		}
		start := time.Now()
		reqID := contextGetRequestID(r)
		app.logger.Info("Incoming Request", "req_id", reqID, "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(srw, r)
		elapsed := time.Since(start)
		if srw.Status >= 500 {
			app.logger.Error("Returning Response", "req_id", reqID, "status", srw.Status, "elapsed", elapsed.String())
		} else {
			app.logger.Info("Returning Response", "req_id", reqID, "status", srw.Status, "elapsed", elapsed.String())
		}
	})
}

func (app *application) methodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			method := r.PostFormValue("_method")
			if method == "" {
				method = r.Header.Get("X-HTTP-Method-Override")
			}
			if method == "PUT" || method == "PATCH" || method == "DELETE" {
				r.Method = method
			}
		}
		next.ServeHTTP(w, r)
	})
}
