package query_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestQueryEndpointSupportsBasicCreateInsertAndSelectFlow(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	createTable := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);",
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:         "write",
			LastInsertID: 0,
			RowsAffected: 0,
		}},
	}, createTable)

	insertRow := server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (name) VALUES ('Ada');",
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:         "write",
			LastInsertID: 1,
			RowsAffected: 1,
		}},
	}, insertRow)

	selectRows := server.Query(t, "", harness.Query{
		Query: "SELECT id, name FROM users;",
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"id", "name"},
			Types:   []string{"INTEGER", "TEXT"},
			Rows:    [][]any{{float64(1), "Ada"}},
		}},
	}, selectRows)
}
