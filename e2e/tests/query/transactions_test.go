package query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestTransactionsCommitMakesChangesVisibleOutsideTheTransaction(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{
			Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT);",
		},
	)

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	require.Equal(t, "begin", begin.Results[0].Type)
	require.NotEmpty(t, begin.Results[0].TxID)
	txID := begin.Results[0].TxID

	insert := server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (name, surname) VALUES ('John', 'Doe');",
		TxID:  txID,
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:         "write",
			LastInsertID: 1,
			RowsAffected: 1,
		}},
	}, insert)

	insideTransaction := server.Query(
		t,
		"",
		harness.Query{Query: "SELECT COUNT(*) FROM users;", TxID: txID},
	)
	require.Equal(t, [][]any{{float64(1)}}, insideTransaction.Results[0].Rows)

	outsideTransaction := server.Query(t, "", harness.Query{Query: "SELECT COUNT(*) FROM users;"})
	require.Equal(t, [][]any{{float64(0)}}, outsideTransaction.Results[0].Rows)

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	require.Equal(t, "commit", commit.Results[0].Type)

	afterCommit := server.Query(t, "", harness.Query{Query: "SELECT COUNT(*) FROM users;"})
	require.Equal(t, [][]any{{float64(1)}}, afterCommit.Results[0].Rows)
}

func TestTransactionsRollbackDiscardsChanges(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{
			Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT);",
		},
	)

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (name, surname) VALUES ('John', 'Doe');",
		TxID:  txID,
	})

	rollback := server.Query(t, "", harness.Query{Query: "ROLLBACK;", TxID: txID})
	require.Equal(t, "rollback", rollback.Results[0].Type)

	afterRollback := server.Query(t, "", harness.Query{Query: "SELECT COUNT(*) FROM users;"})
	require.Equal(t, [][]any{{float64(0)}}, afterRollback.Results[0].Rows)
}

func TestTransactionsRejectInvalidTransactionOperations(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	testCases := []struct {
		name  string
		query harness.Query
	}{
		{name: "commit without begin", query: harness.Query{Query: "COMMIT;"}},
		{name: "rollback without begin", query: harness.Query{Query: "ROLLBACK;"}},
		{
			name:  "commit unknown transaction",
			query: harness.Query{Query: "COMMIT;", TxID: "invalid"},
		},
		{
			name:  "rollback unknown transaction",
			query: harness.Query{Query: "ROLLBACK;", TxID: "invalid"},
		},
		{
			name:  "query unknown transaction",
			query: harness.Query{Query: "SELECT 1, 2, 3;", TxID: "invalid"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := server.Query(t, "", testCase.query)
			require.Equal(t, "error", response.Results[0].Type)
			require.NotEmpty(t, response.Results[0].Error)
		})
	}
}

func TestTransactionsRequireTheExactTransactionID(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"},
	)

	t.Run("commit without txId", func(t *testing.T) {
		response := server.Query(t, "", harness.Query{Query: "COMMIT;"})
		require.Equal(t, harness.QueryResponse{
			Results: []harness.QueryResult{{
				Type:  "error",
				Error: "transaction ID is required for this operation",
			}},
		}, response)
	})

	t.Run("rollback without txId", func(t *testing.T) {
		response := server.Query(t, "", harness.Query{Query: "ROLLBACK;"})
		require.Equal(t, harness.QueryResponse{
			Results: []harness.QueryResult{{
				Type:  "error",
				Error: "transaction ID is required for this operation",
			}},
		}, response)
	})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	t.Run("query with wrong txId", func(t *testing.T) {
		response := server.Query(t, "", harness.Query{Query: "SELECT 1;", TxID: "wrong-tx-id"})
		require.Equal(t, harness.QueryResponse{
			Results: []harness.QueryResult{{
				Type:  "error",
				Error: "transaction ID does not match the currently active transaction",
			}},
		}, response)
	})

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	require.Equal(t, "commit", commit.Results[0].Type)

	t.Run("query after timeout or close reports missing transaction", func(t *testing.T) {
		response := server.Query(t, "", harness.Query{Query: "SELECT 1;", TxID: txID})
		require.Equal(t, harness.QueryResponse{
			Results: []harness.QueryResult{{
				Type:  "error",
				Error: "transaction not found or timed out, check your settings",
			}},
		}, response)
	})
}

func TestTransactionsAcceptEndTransactionAsARollbackAlias(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"},
	)

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (name) VALUES ('Ada');",
		TxID:  txID,
	})

	endTransaction := server.Query(t, "", harness.Query{Query: "END TRANSACTION;", TxID: txID})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type: "rollback",
		}},
	}, endTransaction)

	afterRollback := server.Query(t, "", harness.Query{Query: "SELECT COUNT(*) FROM users;"})
	require.Equal(t, [][]any{{float64(0)}}, afterRollback.Results[0].Rows)
}

func TestTransactionsRollbackIdleTransactionsAutomatically(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{TxIdleTimeout: 100 * time.Millisecond})
	server.Query(
		t,
		"",
		harness.Query{Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"},
	)
	baselineStats := server.Stats(t, "")

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (name) VALUES ('Ada');",
		TxID:  txID,
	})

	require.Eventually(t, func() bool {
		stats := server.Stats(t, "")
		return stats.Totals.Rollbacks == baselineStats.Totals.Rollbacks+1
	}, 2*time.Second, 50*time.Millisecond)

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:  "error",
			Error: "transaction not found or timed out, check your settings",
		}},
	}, commit)

	afterTimeout := server.Query(t, "", harness.Query{Query: "SELECT COUNT(*) FROM users;"})
	require.Equal(t, [][]any{{float64(0)}}, afterTimeout.Results[0].Rows)
}

func TestTransactionsRejectNestedAndClosedTransactionReuse(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	nestedBegin := server.Query(t, "", harness.Query{Query: "BEGIN;", TxID: txID})
	require.Equal(t, "error", nestedBegin.Results[0].Type)
	require.NotEmpty(t, nestedBegin.Results[0].Error)

	server.Query(t, "", harness.Query{Query: "SELECT 1, 2, 3;", TxID: txID})
	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	require.Equal(t, "commit", commit.Results[0].Type)

	afterCommit := server.Query(t, "", harness.Query{Query: "SELECT 1, 2, 3;", TxID: txID})
	require.Equal(t, "error", afterCommit.Results[0].Type)
	require.NotEmpty(t, afterCommit.Results[0].Error)
}
