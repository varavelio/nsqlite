package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitForServer(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		err := WaitForServer(server.URL, 2*time.Second)
		assert.NoError(t, err)
	})

	t.Run("Timeout", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		)
		server.Close()

		err := WaitForServer(server.URL, 1*time.Second)
		assert.Error(t, err)
	})

	t.Run("NotReadyInitially", func(t *testing.T) {
		ready := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ready {
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
			}
		}))
		defer server.Close()

		go func() {
			time.Sleep(500 * time.Millisecond)
			ready = true
		}()

		err := WaitForServer(server.URL, 2*time.Second)
		assert.NoError(t, err)
	})

	t.Run("Non200Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer server.Close()

		err := WaitForServer(server.URL, 1*time.Second)
		assert.Error(t, err)
	})

	t.Run("InvalidURL", func(t *testing.T) {
		invalidURL := "http://invalid-url"

		err := WaitForServer(invalidURL, 1*time.Second)
		assert.Error(t, err)
	})
}
