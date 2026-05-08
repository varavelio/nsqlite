package server

import (
	"net/http"
	"strings"
)

// CORSConfig controls cross-origin request handling for the HTTP server.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

const defaultCORSAllowedMethods = "GET, HEAD, POST, OPTIONS"

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	if s.DisableCORS || len(s.CORS.AllowedOrigins) == 0 {
		return next
	}

	allowedHeaders := strings.Join(s.CORS.AllowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowedOrigin, ok := matchCORSOrigin(s.CORS.AllowedOrigins, origin)
		if !ok {
			if isCORSPreflight(r) {
				http.Error(w, "CORS origin is not allowed", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		headers := w.Header()
		headers.Add("Vary", "Origin")
		headers.Set("Access-Control-Allow-Origin", allowedOrigin)
		if s.CORS.AllowCredentials {
			headers.Set("Access-Control-Allow-Credentials", "true")
		}

		if !isCORSPreflight(r) {
			next.ServeHTTP(w, r)
			return
		}

		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")
		headers.Set("Access-Control-Allow-Methods", defaultCORSAllowedMethods)
		if allowedHeaders != "" {
			headers.Set("Access-Control-Allow-Headers", allowedHeaders)
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func matchCORSOrigin(allowedOrigins []string, origin string) (string, bool) {
	if len(allowedOrigins) == 0 {
		return "", false
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return "*", true
		}
		if origin == allowedOrigin {
			return origin, true
		}
	}

	return "", false
}

func isCORSPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		strings.TrimSpace(r.Header.Get("Origin")) != "" &&
		strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")) != ""
}
