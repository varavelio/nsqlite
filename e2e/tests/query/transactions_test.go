package query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func requireSuccessfulQueryResult(
	t testing.TB,
	response harness.QueryResponse,
	expectedType string,
) harness.QueryResult {
	t.Helper()

	require.Len(t, response.Results, 1)
	result := response.Results[0]
	require.Equalf(t, expectedType, result.Type, "unexpected query error: %s", result.Error)
	require.Empty(t, result.Error)

	return result
}

func calibratedLongRunningReadQuery(
	t testing.TB,
	server *harness.Server,
	minDuration time.Duration,
) (string, float64) {
	t.Helper()

	limits := []int{100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000}
	for _, limit := range limits {
		query := recursiveCounterQuery(limit)
		startedAt := time.Now()
		response := server.Query(t, "", harness.Query{Query: query})
		elapsed := time.Since(startedAt)

		result := requireSuccessfulQueryResult(t, response, "read")
		expected := float64(limit)
		require.Equal(t, [][]any{{expected}}, result.Rows)

		if elapsed >= minDuration {
			return query, expected
		}
	}

	t.Fatalf("failed to calibrate a read query lasting at least %s", minDuration)
	return "", 0
}

func recursiveCounterQuery(limit int) string {
	return fmt.Sprintf(`WITH RECURSIVE counter(value) AS (
  VALUES(0)
  UNION ALL
  SELECT value + 1 FROM counter WHERE value < %d
)
SELECT max(value) FROM counter;`, limit)
}

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

func TestTransactionsAllowDependentDDLSchemaChangesBeforeCommit(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN TRANSACTION;"})
	beginResult := requireSuccessfulQueryResult(t, begin, "begin")
	txID := beginResult.TxID
	require.NotEmpty(t, txID)

	parent := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE parent (id TEXT PRIMARY KEY);",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, parent, "write")

	child := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent(id));",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, child, "write")

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	requireSuccessfulQueryResult(t, commit, "commit")

	schema := server.Query(t, "", harness.Query{
		Query: "SELECT name FROM sqlite_schema WHERE type = 'table' AND name IN ('parent', 'child') ORDER BY name;",
	})
	schemaResult := requireSuccessfulQueryResult(t, schema, "read")
	require.Equal(t, [][]any{{"child"}, {"parent"}}, schemaResult.Rows)
}

func TestTransactionsAllowCommentedDependentDDLBeforeCommit(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	beginResult := requireSuccessfulQueryResult(t, begin, "begin")
	txID := beginResult.TxID
	require.NotEmpty(t, txID)

	parent := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE parent (id TEXT PRIMARY KEY);",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, parent, "write")

	child := server.Query(t, "", harness.Query{
		Query: "-- leading comment before transaction-bound DDL\nCREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent(id));",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, child, "write")

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	requireSuccessfulQueryResult(t, commit, "commit")

	schema := server.Query(t, "", harness.Query{
		Query: "SELECT name FROM sqlite_schema WHERE type = 'table' AND name IN ('parent', 'child') ORDER BY name;",
	})
	schemaResult := requireSuccessfulQueryResult(t, schema, "read")
	require.Equal(t, [][]any{{"child"}, {"parent"}}, schemaResult.Rows)
}

func TestTransactionsRollbackDependentDDLSchemaChangesAtomically(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	beginResult := requireSuccessfulQueryResult(t, begin, "begin")
	txID := beginResult.TxID
	require.NotEmpty(t, txID)

	parent := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE parent (id TEXT PRIMARY KEY);",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, parent, "write")

	child := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent(id));",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, child, "write")

	rollback := server.Query(t, "", harness.Query{Query: "ROLLBACK;", TxID: txID})
	requireSuccessfulQueryResult(t, rollback, "rollback")

	schema := server.Query(t, "", harness.Query{
		Query: "SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name IN ('parent', 'child');",
	})
	schemaResult := requireSuccessfulQueryResult(t, schema, "read")
	require.Equal(t, [][]any{{float64(0)}}, schemaResult.Rows)
}

func TestTransactionsAllowDMLAgainstUncommittedSchemaChanges(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	beginResult := requireSuccessfulQueryResult(t, begin, "begin")
	txID := beginResult.TxID
	require.NotEmpty(t, txID)

	create := server.Query(t, "", harness.Query{
		Query: "CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL);",
		TxID:  txID,
	})
	requireSuccessfulQueryResult(t, create, "write")

	insert := server.Query(t, "", harness.Query{
		Query: "INSERT INTO users (id, email) VALUES ('u1', 'u1@example.com');",
		TxID:  txID,
	})
	insertResult := requireSuccessfulQueryResult(t, insert, "write")
	require.Equal(t, int64(1), insertResult.RowsAffected)

	insideTransaction := server.Query(t, "", harness.Query{
		Query: "SELECT email FROM users WHERE id = 'u1';",
		TxID:  txID,
	})
	insideTransactionResult := requireSuccessfulQueryResult(t, insideTransaction, "read")
	require.Equal(t, [][]any{{"u1@example.com"}}, insideTransactionResult.Rows)

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	requireSuccessfulQueryResult(t, commit, "commit")

	afterCommit := server.Query(
		t,
		"",
		harness.Query{Query: "SELECT email FROM users WHERE id = 'u1';"},
	)
	afterCommitResult := requireSuccessfulQueryResult(t, afterCommit, "read")
	require.Equal(t, [][]any{{"u1@example.com"}}, afterCommitResult.Rows)
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

func TestTransactionsValidateTransactionIDBeforeClassification(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID
	require.NotEmpty(t, txID)

	wrongTxID := server.Query(t, "", harness.Query{
		Query: "SELECT * FROM;",
		TxID:  "wrong-tx-id",
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:  "error",
			Error: "transaction ID does not match the currently active transaction",
		}},
	}, wrongTxID)

	rollback := server.Query(t, "", harness.Query{Query: "ROLLBACK;", TxID: txID})
	require.Equal(t, "rollback", rollback.Results[0].Type)

	closedTxID := server.Query(t, "", harness.Query{
		Query: "SELECT * FROM;",
		TxID:  txID,
	})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:  "error",
			Error: "transaction not found or timed out, check your settings",
		}},
	}, closedTxID)
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

func TestTransactionsDoNotTimeoutWhileAQueryIsRunning(t *testing.T) {
	t.Parallel()

	txIdleTimeout := 50 * time.Millisecond
	server := harness.StartServer(t, harness.ServerConfig{TxIdleTimeout: txIdleTimeout})
	longQuery, expectedMax := calibratedLongRunningReadQuery(t, server, 5*txIdleTimeout)

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	beginResult := requireSuccessfulQueryResult(t, begin, "begin")
	txID := beginResult.TxID
	require.NotEmpty(t, txID)

	longRead := server.Query(t, "", harness.Query{
		Query: longQuery,
		TxID:  txID,
	})
	longReadResult := requireSuccessfulQueryResult(t, longRead, "read")
	require.Equal(t, [][]any{{expectedMax}}, longReadResult.Rows)

	commit := server.Query(t, "", harness.Query{Query: "COMMIT;", TxID: txID})
	requireSuccessfulQueryResult(t, commit, "commit")
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
