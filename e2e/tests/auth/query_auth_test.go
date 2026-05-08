package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestQueryEndpointRejectsRequestsWithoutATokenWhenAuthIsConfigured(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken: "admin-token",
	})

	response := server.PostJSON(t, "/query", []harness.Query{{Query: "SELECT 1;"}}, "")

	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	apiError := harness.DecodeJSON[harness.APIError](t, response)
	require.Equal(t, "Unauthorized", apiError.Error)
	require.Equal(t, "Unauthorized", apiError.Message)
	require.NotEmpty(t, apiError.ID)
}

func TestReadOnlyTokenCanReadButCannotWrite(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})

	readResponse := server.Query(t, "read-only-token", harness.Query{Query: "SELECT 1;"})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"1"},
			Types:   []string{"INTEGER"},
			Rows:    [][]any{{float64(1)}},
		}},
	}, readResponse)

	writeResponse := server.PostJSON(
		t,
		"/query",
		[]harness.Query{{Query: "CREATE TABLE blocked (id INTEGER PRIMARY KEY);"}},
		"read-only-token",
	)
	require.Equal(t, http.StatusForbidden, writeResponse.StatusCode)

	apiError := harness.DecodeJSON[harness.APIError](t, writeResponse)
	require.Equal(t, "Forbidden", apiError.Error)
	require.Equal(t, "Forbidden", apiError.Message)
	require.NotEmpty(t, apiError.ID)

	allowedWrite := server.Query(t, "read-write-token", harness.Query{
		Query: "CREATE TABLE allowed (id INTEGER PRIMARY KEY);",
	})
	require.Len(t, allowedWrite.Results, 1)
	require.Equal(t, "write", allowedWrite.Results[0].Type)
	require.Empty(t, allowedWrite.Results[0].Error)
}

func TestReadOnlyAuthorizationUsesSQLiteClassificationForCTEs(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})
	server.Query(t, "read-write-token", harness.Query{
		Query: "CREATE TABLE cte_auth (id INTEGER PRIMARY KEY);",
	})

	readOnlyCTE := server.Query(t, "read-only-token", harness.Query{
		Query: "WITH value(id) AS (VALUES (1)) SELECT id FROM value;",
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"id"},
			Types:   []string{"INTEGER"},
			Rows:    [][]any{{float64(1)}},
		}},
	}, readOnlyCTE)

	blockedCTEWrite := server.PostJSON(t, "/query", []harness.Query{{
		Query: "WITH value(id) AS (VALUES (1)) INSERT INTO cte_auth (id) SELECT id FROM value;",
	}}, "read-only-token")
	require.Equal(t, http.StatusForbidden, blockedCTEWrite.StatusCode)

	allowedCTEWrite := server.Query(t, "read-write-token", harness.Query{
		Query: "WITH value(id) AS (VALUES (1)) INSERT INTO cte_auth (id) SELECT id FROM value;",
	})
	require.Len(t, allowedCTEWrite.Results, 1)
	require.Equal(t, "write", allowedCTEWrite.Results[0].Type)
	require.Empty(t, allowedCTEWrite.Results[0].Error)
	require.Equal(t, int64(1), allowedCTEWrite.Results[0].RowsAffected)
}
