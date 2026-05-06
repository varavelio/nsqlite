package system_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestStatsEndpointReportsRuntimeFieldsAndCounters(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	baseline := server.Stats(t, "")

	startedAt, err := time.Parse(time.RFC3339, baseline.StartedAt)
	require.NoError(t, err)
	require.False(t, startedAt.IsZero())
	require.NotEmpty(t, baseline.Uptime)

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID
	server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})

	begin = server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID = begin.Results[0].TxID
	server.Query(t, "", harness.Query{Query: "ROLLBACK;", TxID: txID})

	failedQuery := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: "invalid"})
	require.Equal(t, "error", failedQuery.Results[0].Type)
	require.NotEmpty(t, failedQuery.Results[0].Error)

	stats := server.Stats(t, "")

	require.Equal(t, baseline.Totals.Begins+2, stats.Totals.Begins)
	require.Equal(t, baseline.Totals.Commits+1, stats.Totals.Commits)
	require.Equal(t, baseline.Totals.Rollbacks+1, stats.Totals.Rollbacks)
	require.Equal(t, baseline.Totals.Errors+1, stats.Totals.Errors)
	require.Equal(t, baseline.Totals.HTTPRequests+5, stats.Totals.HTTPRequests)

	for i, bucket := range stats.Stats {
		bucketTime, err := time.Parse(time.RFC3339, bucket.Minute)
		require.NoError(t, err)

		if i == 0 {
			continue
		}

		previousBucketTime, err := time.Parse(time.RFC3339, stats.Stats[i-1].Minute)
		require.NoError(t, err)
		require.Truef(
			t,
			previousBucketTime.After(bucketTime),
			"stats buckets must be sorted descending: %s before %s",
			stats.Stats[i-1].Minute,
			bucket.Minute,
		)
	}
}
