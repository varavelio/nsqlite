package query_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestQueryEndpointSupportsInsertReturningRows(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT);",
	})

	response := server.Query(t, "", harness.Query{
		Query: "INSERT INTO test (name) VALUES ('Jane') RETURNING *;",
	})

	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "write",
			Columns: []string{"id", "name"},
			Types:   []string{"INTEGER", "TEXT"},
			Rows:    [][]any{{float64(1), "Jane"}},
		}},
	}, response)
}

func TestQueryEndpointAcceptsQueriesWithAndWithoutSemicolons(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	expected := harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"1", "2", "3"},
			Types:   []string{"INTEGER", "INTEGER", "INTEGER"},
			Rows:    [][]any{{float64(1), float64(2), float64(3)}},
		}},
	}

	withSemicolon := server.Query(t, "", harness.Query{Query: "SELECT 1, 2, 3;"})
	withoutSemicolon := server.Query(t, "", harness.Query{Query: "SELECT 1, 2, 3"})

	require.Equal(t, expected, withSemicolon)
	require.Equal(t, expected, withoutSemicolon)
}

func TestQueryEndpointReturnsStructuredErrorsForInvalidQueries(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	queries := []string{
		"SELECT * FROM non_existent_table;",
		"INSERT INTO;",
		"abc",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			response := server.Query(t, "", harness.Query{Query: query})
			require.Len(t, response.Results, 1)
			require.Equal(t, "error", response.Results[0].Type)
			require.NotEmpty(t, response.Results[0].Error)
		})
	}
}
