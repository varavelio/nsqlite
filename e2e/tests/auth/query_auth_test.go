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
