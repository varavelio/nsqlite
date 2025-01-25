package tests

import (
	"io"
	"net/http"
	"testing"

	"github.com/nsqlite/nsqlite/internal/nsqlited/server"
	"github.com/nsqlite/nsqlite/internal/version"
	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	t.Run("Server healthcheck", func(t *testing.T) {
		url := createServer(t) + "/health"

		resp, err := http.Get(url)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, resp.StatusCode, http.StatusOK)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(body), "OK")
	})

	t.Run("Server version", func(t *testing.T) {
		url := createServer(t) + "/version"

		resp, err := http.Get(url)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, resp.StatusCode, http.StatusOK)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(body), version.Version)
	})

	t.Run("Basic query with and without semicolon", func(t *testing.T) {
		url := createServer(t) + "/query"

		responseSemicolon := sendQuery(t, url, server.Query{
			Query: "SELECT 1, 2, 3;",
		})

		responseNoSemicolon := sendQuery(t, url, server.Query{
			Query: "SELECT 1, 2, 3",
		})

		expected := server.Response{
			Results: []server.ResponseResult{
				{
					Type:    "read",
					Columns: []string{"1", "2", "3"},
					Types:   []string{"INTEGER", "INTEGER", "INTEGER"},
					Rows:    [][]any{{float64(1), float64(2), float64(3)}},
				},
			},
		}

		assert.Equal(t, responseSemicolon, expected)
		assert.Equal(t, responseNoSemicolon, expected)
	})
}
