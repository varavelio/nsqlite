package query_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestTransactionCommitPersistsValidChangesAfterConstraintViolation(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE pk_test (id INTEGER PRIMARY KEY, val TEXT);",
	})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	firstInsert := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO pk_test (id, val) VALUES (1, 'first');",
	})
	require.Equal(t, "write", firstInsert.Results[0].Type)

	duplicateInsert := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO pk_test (id, val) VALUES (1, 'duplicate');",
	})
	require.Equal(t, "error", duplicateInsert.Results[0].Type)
	require.Contains(t, duplicateInsert.Results[0].Error, "constraint")
	require.Contains(t, duplicateInsert.Results[0].Error, "failed")

	commit := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "COMMIT;",
	})
	require.Equal(t, "commit", commit.Results[0].Type)

	result := server.Query(t, "", harness.Query{
		Query: "SELECT id, val FROM pk_test;",
	})
	require.Equal(t, [][]any{{float64(1), "first"}}, result.Results[0].Rows)
}

func TestTransactionSyntaxErrorPersistsValidChangesAndCommitSucceeds(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE syntax_test (id INTEGER PRIMARY KEY, val TEXT);",
	})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	validInsert := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO syntax_test (id, val) VALUES (1, 'valid');",
	})
	require.Equal(t, "write", validInsert.Results[0].Type)

	invalidQuery := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO syntax_test VALUES (,,,);",
	})
	require.Equal(t, "error", invalidQuery.Results[0].Type)
	require.Contains(t, invalidQuery.Results[0].Error, "syntax")

	commit := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "COMMIT;",
	})
	require.Equal(t, "commit", commit.Results[0].Type)

	result := server.Query(t, "", harness.Query{
		Query: "SELECT COUNT(*) FROM syntax_test;",
	})
	require.Equal(t, [][]any{{float64(1)}}, result.Results[0].Rows)
}

func TestTransactionErrorAndRollbackDiscardsAllChanges(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE rollback_test (id INTEGER PRIMARY KEY, val TEXT);",
	})

	begin := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID

	server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO rollback_test (id, val) VALUES (1, 'valid-before-error');",
	})

	server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO rollback_test (id, val) VALUES (1, 'duplicate-pk');",
	})

	rollback := server.Query(t, "", harness.Query{
		TxID:  txID,
		Query: "ROLLBACK;",
	})
	require.Equal(t, "rollback", rollback.Results[0].Type)

	result := server.Query(t, "", harness.Query{
		Query: "SELECT COUNT(*) FROM rollback_test;",
	})
	require.Equal(t, [][]any{{float64(0)}}, result.Results[0].Rows)
}

func TestNonTransactionErrorDoesNotRollBackPriorAutoCommittedWrites(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE auto_batch (id INTEGER PRIMARY KEY, val TEXT);",
	})

	firstInsert := server.Query(t, "", harness.Query{
		Query: "INSERT INTO auto_batch (id, val) VALUES (1, 'persisted');",
	})
	require.Equal(t, "write", firstInsert.Results[0].Type)

	duplicateInsert := server.Query(t, "", harness.Query{
		Query: "INSERT INTO auto_batch (id, val) VALUES (1, 'duplicate');",
	})
	require.Equal(t, "error", duplicateInsert.Results[0].Type)
	require.Contains(t, duplicateInsert.Results[0].Error, "constraint")

	result := server.Query(t, "", harness.Query{
		Query: "SELECT id, val FROM auto_batch;",
	})
	require.Equal(t, [][]any{{float64(1), "persisted"}}, result.Results[0].Rows)
}
