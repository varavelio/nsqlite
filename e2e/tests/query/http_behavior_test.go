package query_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestReadOnlyBatchStopsOnForbiddenQueryWithoutExecutingRemainingQueries(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRO: "read-only-token",
	})

	before := server.Stats(t, "admin-token")

	response := server.PostJSON(t, "/query", []harness.Query{
		{Query: "SELECT 1;"},
		{Query: "CREATE TABLE blocked (id INTEGER PRIMARY KEY);"},
		{Query: "SELECT 2;"},
	}, "read-only-token")
	require.Equal(t, http.StatusForbidden, response.StatusCode)

	apiError := harness.DecodeJSON[harness.APIError](t, response)
	require.Equal(t, "Forbidden", apiError.Error)
	require.Equal(t, "Forbidden", apiError.Message)

	after := server.Stats(t, "admin-token")
	require.Equal(t, before.Totals.Reads+1, after.Totals.Reads)
	require.Equal(t, before.Totals.Writes, after.Totals.Writes)

	blockedTable := server.Query(t, "admin-token", harness.Query{
		Query: "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'blocked';",
	})
	require.Equal(t, [][]any{{float64(0)}}, blockedTable.Results[0].Rows)
}
