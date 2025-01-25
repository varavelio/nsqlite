package nsqlited_tests

import (
	"io"
	"net/http"
	"testing"

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

	t.Run("Basic operations", func(t *testing.T) {
		url := createServer(t) + "/query"

		assertQuery(
			t, url,
			Query{
				Query: "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT);",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:         "write",
						LastInsertID: 0,
						RowsAffected: 0,
					},
				},
			},
		)

		assertQuery(
			t, url,
			Query{
				Query: "INSERT INTO test (name) VALUES ('John');",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:         "write",
						LastInsertID: 1,
						RowsAffected: 1,
					},
				},
			},
		)

		assertQuery(
			t, url,
			Query{
				Query: "INSERT INTO test (name) VALUES ('Jane') RETURNING *;",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:    "write",
						Columns: []string{"id", "name"},
						Types:   []string{"INTEGER", "TEXT"},
						Rows:    [][]any{{float64(2), "Jane"}},
					},
				},
			},
		)

		assertQuery(
			t, url,
			Query{
				Query: "SELECT * FROM test;",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:    "read",
						Columns: []string{"id", "name"},
						Types:   []string{"INTEGER", "TEXT"},
						Rows:    [][]any{{float64(1), "John"}, {float64(2), "Jane"}},
					},
				},
			},
		)

		assertQuery(
			t, url,
			Query{
				Query: "DELETE FROM test;",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:         "write",
						LastInsertID: 2,
						RowsAffected: 2,
					},
				},
			},
		)

		assertQuery(
			t, url,
			Query{
				Query: "SELECT * FROM test;",
			},
			Response{
				Results: []ResponseResult{
					{
						Type:    "read",
						Columns: []string{"id", "name"},
						Types:   []string{"INTEGER", "TEXT"},
					},
				},
			},
		)
	})

	t.Run("Query with and without semicolon", func(t *testing.T) {
		url := createServer(t) + "/query"

		expected := Response{
			Results: []ResponseResult{
				{
					Type:    "read",
					Columns: []string{"1", "2", "3"},
					Types:   []string{"INTEGER", "INTEGER", "INTEGER"},
					Rows:    [][]any{{float64(1), float64(2), float64(3)}},
				},
			},
		}

		assertQuery(t, url, Query{
			Query: "SELECT 1, 2, 3;",
		}, expected)

		assertQuery(t, url, Query{
			Query: "SELECT 1, 2, 3",
		}, expected)
	})
}
