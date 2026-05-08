package query_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestReadOnlyBatchReturnsSQLiteErrorAndContinuesWithSafeQueries(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRO: "read-only-token",
	})

	before := server.Stats(t, "admin-token")

	response := server.QueryResponse(t, "read-only-token",
		harness.Query{Query: "SELECT 1;"},
		harness.Query{Query: "CREATE TABLE blocked (id INTEGER PRIMARY KEY);"},
		harness.Query{Query: "SELECT 2;"},
	)
	require.Equal(t, http.StatusOK, response.StatusCode)

	queryResponse := harness.DecodeQueryResponse(t, response).WithoutTiming()
	require.Len(t, queryResponse.Results, 3)
	require.Equal(t, "read", queryResponse.Results[0].Type)
	require.Equal(t, [][]any{{float64(1)}}, queryResponse.Results[0].Rows)
	require.Equal(t, "error", queryResponse.Results[1].Type)
	require.Contains(t, queryResponse.Results[1].Error, "23: authorization denied")
	require.Equal(t, "read", queryResponse.Results[2].Type)
	require.Equal(t, [][]any{{float64(2)}}, queryResponse.Results[2].Rows)

	after := server.Stats(t, "admin-token")
	require.Equal(t, before.Totals.Reads+2, after.Totals.Reads)
	require.Equal(t, before.Totals.Writes, after.Totals.Writes)

	blockedTable := server.Query(t, "admin-token", harness.Query{
		Query: "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'blocked';",
	})
	require.Equal(t, [][]any{{float64(0)}}, blockedTable.Results[0].Rows)
}
