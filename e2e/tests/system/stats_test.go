package system_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestStatsEndpointReportsEmptyServerTotals(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	stats := server.Stats(t, "")

	require.NotEmpty(t, stats.StartedAt)
	require.NotEmpty(t, stats.Uptime)
	require.Equal(t, harness.LoadedStats{
		QueuedBegins:       0,
		QueuedWrites:       0,
		QueuedHTTPRequests: 0,
		Totals: harness.Totals{
			Reads:        1,
			Writes:       0,
			Begins:       0,
			Commits:      0,
			Rollbacks:    0,
			Errors:       0,
			HTTPRequests: 0,
		},
		Stats: stats.Stats,
	}, stats.WithoutRuntime())
}

func TestStatsEndpointCountsReadAndWriteQueries(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{Query: "SELECT 1;"})
	statsAfterRead := server.Stats(t, "")

	require.Equal(t, int64(2), statsAfterRead.Totals.Reads)
	require.Equal(t, int64(0), statsAfterRead.Totals.Writes)
	require.Equal(t, int64(1), statsAfterRead.Totals.HTTPRequests)

	server.Query(
		t,
		"",
		harness.Query{Query: "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT);"},
	)
	statsAfterWrite := server.Stats(t, "")

	require.Equal(t, int64(2), statsAfterWrite.Totals.Reads)
	require.Equal(t, int64(1), statsAfterWrite.Totals.Writes)
	require.Equal(t, int64(2), statsAfterWrite.Totals.HTTPRequests)
	require.Equal(t, int64(0), statsAfterWrite.QueuedBegins)
	require.Equal(t, int64(0), statsAfterWrite.QueuedWrites)
	require.Equal(t, int64(0), statsAfterWrite.QueuedHTTPRequests)
}
