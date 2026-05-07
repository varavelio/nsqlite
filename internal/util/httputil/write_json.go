package httputil

import (
	"fmt"
	"net/http"

	"github.com/goccy/go-json"
)

// WriteJSON writes a JSON response to the given http.ResponseWriter.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("failed to write JSON response: %w", err)
	}

	return nil
}

// WriteJSONBytes writes a byte slice as a JSON response to the given http.ResponseWriter.
func WriteJSONBytes(w http.ResponseWriter, status int, b []byte) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("failed to write JSON response: %w", err)
	}

	return nil
}
