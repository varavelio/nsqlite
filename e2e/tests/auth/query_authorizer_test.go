package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestReadWriteTokenCannotRunDDLBuCanRunDML(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT);",
	})

	insert := server.Query(t, "read-write-token", harness.Query{
		Query: "INSERT INTO projects (name) VALUES ('alpha');",
	})
	require.Equal(t, "write", insert.Results[0].Type)
	require.Equal(t, int64(1), insert.Results[0].RowsAffected)

	update := server.Query(t, "read-write-token", harness.Query{
		Query: "UPDATE projects SET name = 'beta' WHERE id = 1;",
	})
	require.Equal(t, "write", update.Results[0].Type)
	require.Equal(t, int64(1), update.Results[0].RowsAffected)

	for _, ddl := range []string{
		"ALTER TABLE projects ADD COLUMN description TEXT;",
		"DROP TABLE projects;",
	} {
		t.Run(ddl, func(t *testing.T) {
			response := server.Query(t, "read-write-token", harness.Query{Query: ddl})
			require.Equal(t, "error", response.Results[0].Type)
			require.Contains(t, response.Results[0].Error, "23: authorization denied")
		})
	}

	readBack := server.Query(t, "read-write-token", harness.Query{
		Query: "SELECT id, name FROM projects;",
	})
	require.Equal(t, [][]any{{float64(1), "beta"}}, readBack.Results[0].Rows)
}

func TestAdminTokenCanRunDDLWithoutRestrictions(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{AuthToken: "admin-token"})

	create := server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE admin_managed (id INTEGER PRIMARY KEY, name TEXT);",
	})
	require.Equal(t, "write", create.Results[0].Type)

	alter := server.Query(t, "admin-token", harness.Query{
		Query: "ALTER TABLE admin_managed ADD COLUMN created_at TEXT;",
	})
	require.Equal(t, "write", alter.Results[0].Type)

	drop := server.Query(t, "admin-token", harness.Query{
		Query: "DROP TABLE admin_managed;",
	})
	require.Equal(t, "write", drop.Results[0].Type)
}

func TestReadOnlyTokenCannotEscalateThroughTransactions(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRO: "read-only-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE readonly_guard (id INTEGER PRIMARY KEY, name TEXT);",
	})

	beginResponse := server.PostJSON(
		t,
		"/query",
		[]harness.Query{{Query: "BEGIN;"}},
		"read-only-token",
	)
	require.Equal(t, http.StatusOK, beginResponse.StatusCode)
	beginResult := harness.DecodeJSON[harness.QueryResponse](t, beginResponse).WithoutTiming()
	require.Equal(t, "error", beginResult.Results[0].Type)
	require.Contains(t, beginResult.Results[0].Error, "23: authorization denied")

	adminBegin := server.Query(t, "admin-token", harness.Query{Query: "BEGIN;"})
	txID := adminBegin.Results[0].TxID
	require.NotEmpty(t, txID)

	roInsertResponse := server.PostJSON(t, "/query", []harness.Query{{
		TxID:  txID,
		Query: "INSERT INTO readonly_guard (name) VALUES ('blocked');",
	}}, "read-only-token")
	require.Equal(t, http.StatusOK, roInsertResponse.StatusCode)
	roInsert := harness.DecodeJSON[harness.QueryResponse](t, roInsertResponse).WithoutTiming()
	require.Equal(t, "error", roInsert.Results[0].Type)
	require.Equal(
		t,
		"transaction ID does not match the currently active transaction",
		roInsert.Results[0].Error,
	)

	rollback := server.Query(t, "admin-token", harness.Query{Query: "ROLLBACK;", TxID: txID})
	require.Equal(t, "rollback", rollback.Results[0].Type)

	readCheck := server.Query(t, "read-only-token", harness.Query{
		Query: "SELECT COUNT(*) FROM readonly_guard;",
	})
	require.Equal(t, [][]any{{float64(0)}}, readCheck.Results[0].Rows)
}

func TestAuthorizerRoleDoesNotLeakBetweenRequests(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE role_reset (id INTEGER PRIMARY KEY, name TEXT);",
	})

	rwInsert := server.Query(t, "read-write-token", harness.Query{
		Query: "INSERT INTO role_reset (name) VALUES ('first');",
	})
	require.Equal(t, "write", rwInsert.Results[0].Type)

	adminAlter := server.Query(t, "admin-token", harness.Query{
		Query: "ALTER TABLE role_reset ADD COLUMN status TEXT;",
	})
	require.Equal(t, "write", adminAlter.Results[0].Type)

	rwUpdate := server.Query(t, "read-write-token", harness.Query{
		Query: "UPDATE role_reset SET status = 'active' WHERE id = 1;",
	})
	require.Equal(t, "write", rwUpdate.Results[0].Type)

	rwDrop := server.Query(t, "read-write-token", harness.Query{
		Query: "DROP TABLE role_reset;",
	})
	require.Equal(t, "error", rwDrop.Results[0].Type)
	require.Contains(t, rwDrop.Results[0].Error, "23: authorization denied")

	adminDrop := server.Query(t, "admin-token", harness.Query{
		Query: "DROP TABLE role_reset;",
	})
	require.Equal(t, "write", adminDrop.Results[0].Type)
}

func TestTransactionTxIDCannotBeUsedByAnotherPrincipal(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "rw-token-a rw-token-b",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE tx_owner_guard (id INTEGER PRIMARY KEY, name TEXT);",
	})

	begin := server.Query(t, "rw-token-a", harness.Query{Query: "BEGIN;"})
	txID := begin.Results[0].TxID
	require.NotEmpty(t, txID)

	insert := server.Query(t, "rw-token-a", harness.Query{
		TxID:  txID,
		Query: "INSERT INTO tx_owner_guard (name) VALUES ('owner-write');",
	})
	require.Equal(t, "write", insert.Results[0].Type)

	hijackResponse := server.PostJSON(t, "/query", []harness.Query{{
		TxID:  txID,
		Query: "SELECT id, name FROM tx_owner_guard;",
	}}, "rw-token-b")
	require.Equal(t, http.StatusOK, hijackResponse.StatusCode)
	hijack := harness.DecodeJSON[harness.QueryResponse](t, hijackResponse).WithoutTiming()
	require.Equal(t, "error", hijack.Results[0].Type)
	require.Equal(
		t,
		"transaction ID does not match the currently active transaction",
		hijack.Results[0].Error,
	)

	rollback := server.Query(t, "rw-token-a", harness.Query{Query: "ROLLBACK;", TxID: txID})
	require.Equal(t, "rollback", rollback.Results[0].Type)

	readBack := server.Query(t, "rw-token-a", harness.Query{
		Query: "SELECT COUNT(*) FROM tx_owner_guard;",
	})
	require.Equal(t, [][]any{{float64(0)}}, readBack.Results[0].Rows)
}
