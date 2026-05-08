package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/internal/logger"
)

func TestCORSMiddleware(t *testing.T) {
	t.Run("handles configured preflight requests", func(t *testing.T) {
		server := newTestServer(t, Config{
			Logger: logger.NewLogger(),
			CORS: CORSConfig{
				AllowedOrigins:   []string{"https://console.example.com"},
				AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Trace-ID"},
				AllowCredentials: true,
			},
		})

		handler := server.corsMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-ID", "req-123")
				w.WriteHeader(http.StatusOK)
			}),
		)

		preflightRequest := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodOptions,
			"/rpc/Database/query",
			nil,
		)
		preflightRequest.Header.Set("Origin", "https://console.example.com")
		preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodPost)
		preflightRequest.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

		preflightResponse := httptest.NewRecorder()
		handler.ServeHTTP(preflightResponse, preflightRequest)

		require.Equal(t, http.StatusNoContent, preflightResponse.Code)
		require.Equal(
			t,
			"https://console.example.com",
			preflightResponse.Header().Get("Access-Control-Allow-Origin"),
		)
		require.Equal(
			t,
			"GET, HEAD, POST, OPTIONS",
			preflightResponse.Header().Get("Access-Control-Allow-Methods"),
		)
		require.Equal(
			t,
			"Authorization, Content-Type, X-Trace-ID",
			preflightResponse.Header().Get("Access-Control-Allow-Headers"),
		)
		require.Equal(t, "true", preflightResponse.Header().Get("Access-Control-Allow-Credentials"))

		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Set("Origin", "https://console.example.com")

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(
			t,
			"https://console.example.com",
			response.Header().Get("Access-Control-Allow-Origin"),
		)
		require.Empty(t, response.Header().Get("Access-Control-Expose-Headers"))
	})

	t.Run("rejects blocked origins during preflight", func(t *testing.T) {
		server := newTestServer(t, Config{
			Logger: logger.NewLogger(),
			CORS: CORSConfig{
				AllowedOrigins: []string{"https://console.example.com"},
			},
		})

		request := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodOptions,
			"/rpc/Database/query",
			nil,
		)
		request.Header.Set("Origin", "https://blocked.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)

		response := httptest.NewRecorder()
		server.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(response, request)

		require.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestNewServer(t *testing.T) {
	t.Run("returns an error when CORS is enabled without allowed origins", func(t *testing.T) {
		_, err := NewServer(Config{Logger: logger.NewLogger()})

		require.EqualError(t, err, "cors allowed origins must contain at least one origin")
	})

	t.Run("allows configured CORS origins", func(t *testing.T) {
		server, err := NewServer(Config{
			Logger: logger.NewLogger(),
			CORS: CORSConfig{
				AllowedOrigins: []string{"*"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, server)
	})
}

func newTestServer(t *testing.T, cfg Config) *Server {
	t.Helper()

	server, err := NewServer(cfg)
	require.NoError(t, err)
	return server
}
