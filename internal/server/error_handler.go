package server

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/varavelio/nsqlite/internal/log"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

func (s *Server) errorHandler(
	w http.ResponseWriter, r *http.Request, err error,
) {
	ip := httputil.ReadUserIP(r)
	errorURL := r.URL.String()
	errorId := uuid.NewString()

	if jsonErr, ok := errors.AsType[httputil.JSONError](err); ok {
		statusText := http.StatusText(jsonErr.HTTPStatus)
		safeMessage := jsonErr.SafeMessage
		if safeMessage == "" {
			safeMessage = statusText
		}

		s.Logger.ErrorNs(
			log.NsServer, "error while handling request", log.KV{
				"id":      errorId,
				"status":  jsonErr.HTTPStatus,
				"error":   jsonErr.Error(),
				"message": safeMessage,
				"url":     errorURL,
				"ip":      ip,
			},
		)

		_ = httputil.WriteJSON(w, jsonErr.HTTPStatus, map[string]any{
			"id":      errorId,
			"error":   statusText,
			"message": safeMessage,
		})
	} else {
		s.Logger.ErrorNs(
			log.NsServer, "unknown error while handling request", log.KV{
				"id":    errorId,
				"error": err.Error(),
				"url":   errorURL,
				"ip":    ip,
			},
		)

		_ = httputil.WriteString(
			w, http.StatusInternalServerError, "Internal Server Error - "+errorId,
		)
	}
}
