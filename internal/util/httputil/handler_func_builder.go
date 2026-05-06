package httputil

import (
	"net/http"
	"slices"
)

// HandlerFuncErr behaves like http.HandlerFunc but returns an error.
type HandlerFuncErr func(w http.ResponseWriter, r *http.Request) error

// Middleware wraps a HandlerFuncErr and returns a new one.
type Middleware func(next HandlerFuncErr) HandlerFuncErr

// ErrorHandler handles errors returned by handlers or middlewares.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// HandlerFuncBuilder builds an http.HandlerFunc from a HandlerFuncErr and middlewares.
type HandlerFuncBuilder func(handler HandlerFuncErr, middlewares ...Middleware) http.HandlerFunc

// CreateHandlerFuncBuilder returns a function that creates an http.HandlerFunc
// by chaining middlewares and a final handler, using a centralized error handler.
func CreateHandlerFuncBuilder(errorHandler ErrorHandler) HandlerFuncBuilder {
	return func(handler HandlerFuncErr, middlewares ...Middleware) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			finalHandler := handler

			// Apply middlewares in reverse order for better readability
			// and clarity of the request flow.
			for _, middleware := range slices.Backward(middlewares) {
				previousHandler := finalHandler

				finalHandler = func(writer http.ResponseWriter, request *http.Request) error {
					return middleware(previousHandler)(writer, request)
				}
			}

			// Execute the final handler and handle the error if any
			if err := finalHandler(w, r); err != nil {
				errorHandler(w, r, err)
			}
		}
	}
}
