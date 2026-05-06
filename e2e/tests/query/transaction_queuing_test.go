package query_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestTransactionsQueueRequestsWhenAnotherTransactionIsActive(t *testing.T) {
	server := harness.StartServer(t, harness.ServerConfig{})
	baseline := server.Stats(t, "")

	firstTransaction := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	require.Equal(t, "begin", firstTransaction.Results[0].Type)
	txID1 := firstTransaction.Results[0].TxID
	require.NotEmpty(t, txID1)

	var secondTransaction harness.QueryResponse
	var wg sync.WaitGroup
	wg.Go(func() {
		secondTransaction = server.Query(t, "", harness.Query{Query: "BEGIN;"})
	})

	require.Eventually(t, func() bool {
		stats := server.Stats(t, "")
		return stats.QueuedBegins >= 1
	}, 2*time.Second, 10*time.Millisecond)

	server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID1})

	wg.Wait()

	require.Equal(t, "begin", secondTransaction.Results[0].Type)
	txID2 := secondTransaction.Results[0].TxID
	require.NotEmpty(t, txID2)
	require.NotEqual(t, txID1, txID2)

	server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID2})

	final := server.Stats(t, "")
	require.Equal(t, baseline.Totals.Begins+2, final.Totals.Begins)
	require.Equal(t, baseline.Totals.Commits+2, final.Totals.Commits)
	require.Zero(t, final.QueuedBegins)
	require.Zero(t, final.QueuedWrites)
	require.Zero(t, final.QueuedHTTPRequests)
}
