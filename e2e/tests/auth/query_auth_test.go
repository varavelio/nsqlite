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

	response := server.QueryResponse(t, "", harness.Query{Query: "SELECT 1;"})

	require.Equal(t, http.StatusOK, response.StatusCode)
	rpcError := harness.DecodeJSON[harness.RPCResponse[map[string]any]](t, response)
	require.False(t, rpcError.OK)
	require.Equal(t, "Unauthorized", rpcError.Error.Message)
}

func TestReadOnlyTokenCanReadButCannotWrite(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE allowed (id INTEGER PRIMARY KEY, name TEXT);",
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

	writeResponse := server.QueryResponse(
		t,
		"read-only-token",
		harness.Query{Query: "INSERT INTO allowed (name) VALUES ('blocked');"},
	)
	require.Equal(t, http.StatusOK, writeResponse.StatusCode)

	blockedWrite := harness.DecodeQueryResponse(t, writeResponse).WithoutTiming()
	require.Equal(t, "error", blockedWrite.Results[0].Type)
	require.Contains(t, blockedWrite.Results[0].Error, "23: authorization denied")

	allowedWrite := server.Query(t, "read-write-token", harness.Query{
		Query: "INSERT INTO allowed (name) VALUES ('allowed');",
	})
	require.Len(t, allowedWrite.Results, 1)
	require.Equal(t, "write", allowedWrite.Results[0].Type)
	require.Empty(t, allowedWrite.Results[0].Error)
	require.Equal(t, int64(1), allowedWrite.Results[0].RowsAffected)
}

func TestReadOnlyAuthorizationUsesSQLiteClassificationForCTEs(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})
	server.Query(t, "admin-token", harness.Query{
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

	blockedCTEWrite := server.QueryResponse(t, "read-only-token", harness.Query{
		Query: "WITH value(id) AS (VALUES (1)) INSERT INTO cte_auth (id) SELECT id FROM value;",
	})
	require.Equal(t, http.StatusOK, blockedCTEWrite.StatusCode)
	blockedCTE := harness.DecodeQueryResponse(t, blockedCTEWrite).WithoutTiming()
	require.Equal(t, "error", blockedCTE.Results[0].Type)
	require.Contains(t, blockedCTE.Results[0].Error, "23: authorization denied")

	allowedCTEWrite := server.Query(t, "read-write-token", harness.Query{
		Query: "WITH value(id) AS (VALUES (1)) INSERT INTO cte_auth (id) SELECT id FROM value;",
	})
	require.Len(t, allowedCTEWrite.Results, 1)
	require.Equal(t, "write", allowedCTEWrite.Results[0].Type)
	require.Empty(t, allowedCTEWrite.Results[0].Error)
	require.Equal(t, int64(1), allowedCTEWrite.Results[0].RowsAffected)
}
