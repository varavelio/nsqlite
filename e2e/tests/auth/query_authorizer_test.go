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

func TestAuthorizerFunctionWhitelistViaHTTP(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})

	allowed := []struct {
		name  string
		query string
	}{
		{name: "coalesce", query: "SELECT COALESCE(NULL, 'fallback')"},
		{name: "lower_upper", query: "SELECT LOWER(UPPER('Test'))"},
		{name: "substr", query: "SELECT SUBSTR('hello', 2, 3)"},
		{name: "length", query: "SELECT LENGTH('abc')"},
		{name: "trim", query: "SELECT TRIM('  x  ')"},
		{name: "replace", query: "SELECT REPLACE('abc', 'b', 'x')"},
		{name: "abs", query: "SELECT ABS(-42)"},
		{name: "round", query: "SELECT ROUND(3.14159, 2)"},
		{name: "typeof", query: "SELECT TYPEOF(42)"},
		{name: "iif", query: "SELECT IIF(1 > 0, 'ok', 'no')"},
		{name: "nullif", query: "SELECT NULLIF(1, 1)"},
		{name: "ifnull", query: "SELECT IFNULL(NULL, 'fallback')"},
		{name: "printf", query: "SELECT PRINTF('val=%d', 42)"},
		{name: "date_time", query: "SELECT DATE('now'), TIME('now'), DATETIME('now')"},
		{name: "strftime", query: "SELECT STRFTIME('%Y', 'now')"},
		{name: "json_extract", query: "SELECT JSON_EXTRACT('{\"a\":1}', '$.a')"},
		{name: "json_type", query: "SELECT JSON_TYPE('{\"a\":1}')"},
		{name: "json_valid", query: "SELECT JSON_VALID('{\"a\":1}')"},
		{name: "random", query: "SELECT TYPEOF(RANDOM())"},
		{name: "sqlite_version", query: "SELECT TYPEOF(SQLITE_VERSION())"},
		{name: "last_insert_rowid", query: "SELECT LAST_INSERT_ROWID()"},
	}

	for _, tc := range allowed {
		t.Run("rw_"+tc.name, func(t *testing.T) {
			resp := server.Query(t, "read-write-token", harness.Query{Query: tc.query})
			require.Equal(t, "read", resp.Results[0].Type,
				"function test %q should succeed for RW", tc.name)
		})
		t.Run("ro_"+tc.name, func(t *testing.T) {
			resp := server.Query(t, "read-only-token", harness.Query{Query: tc.query})
			require.Equal(t, "read", resp.Results[0].Type,
				"function test %q should succeed for RO", tc.name)
		})
	}
}

func TestAuthorizerPRAGMAWhitelistViaHTTP(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})

	allowedPragmas := []string{
		"PRAGMA table_info(sqlite_master)",
		"PRAGMA table_xinfo(sqlite_master)",
		"PRAGMA database_list",
		"PRAGMA compile_options",
		"PRAGMA schema_version",
		"PRAGMA user_version",
		"PRAGMA collation_list",
		"PRAGMA function_list",
		"PRAGMA pragma_list",
		"PRAGMA page_count",
		"PRAGMA page_size",
		"PRAGMA data_version",
		"PRAGMA defer_foreign_keys",
		"PRAGMA foreign_key_check(sqlite_master)",
	}

	for _, pragma := range allowedPragmas {
		t.Run("rw_"+pragma, func(t *testing.T) {
			resp := server.Query(t, "read-write-token", harness.Query{Query: pragma})
			require.Equal(t, "read", resp.Results[0].Type,
				"pragma %q should work for RW", pragma)
		})
		t.Run("ro_"+pragma, func(t *testing.T) {
			resp := server.Query(t, "read-only-token", harness.Query{Query: pragma})
			require.Equal(t, "read", resp.Results[0].Type,
				"pragma %q should work for RO", pragma)
		})
	}

	blockedPragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA cache_size = -2000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA writable_schema = ON",
		"PRAGMA wal_checkpoint",
	}

	for _, pragma := range blockedPragmas {
		t.Run("rw_"+pragma, func(t *testing.T) {
			resp := server.Query(t, "read-write-token", harness.Query{Query: pragma})
			require.Equal(t, "error", resp.Results[0].Type)
			require.Contains(t, resp.Results[0].Error, "23: authorization denied",
				"pragma %q must be blocked for RW", pragma)
		})
		t.Run("ro_"+pragma, func(t *testing.T) {
			resp := server.Query(t, "read-only-token", harness.Query{Query: pragma})
			require.Equal(t, "error", resp.Results[0].Type)
			require.Contains(t, resp.Results[0].Error, "23: authorization denied",
				"pragma %q must be blocked for RO", pragma)
		})
	}

	t.Run("admin can set pragma (basic)", func(t *testing.T) {
		resp := server.Query(t, "admin-token", harness.Query{Query: "PRAGMA schema_version"})
		require.Equal(t, "read", resp.Results[0].Type)
	})
}

func TestAuthorizerDDLEdgeCasesViaHTTP(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE ddl_base (id INTEGER PRIMARY KEY, name TEXT);",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE INDEX idx_ddl_name ON ddl_base(name)",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE VIEW ddl_view AS SELECT id, name FROM ddl_base",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TRIGGER ddl_trg AFTER INSERT ON ddl_base BEGIN SELECT 1; END",
	})

	blockedDDL := []string{
		"CREATE INDEX idx_ddl_name2 ON ddl_base(name)",
		"DROP INDEX idx_ddl_name",
		"CREATE VIEW ddl_view2 AS SELECT id, name FROM ddl_base",
		"DROP VIEW ddl_view",
		"CREATE TRIGGER ddl_trg2 AFTER INSERT ON ddl_base BEGIN SELECT 1; END",
		"DROP TRIGGER ddl_trg",
		"ANALYZE ddl_base",
		"REINDEX idx_ddl_name",
		"ATTACH DATABASE ':memory:' AS aux_blocked",
	}

	for _, ddl := range blockedDDL {
		t.Run(ddl, func(t *testing.T) {
			resp := server.Query(t, "read-write-token", harness.Query{Query: ddl})
			require.Equal(t, "error", resp.Results[0].Type,
				"DDL %q should be blocked for RW", ddl)
			require.Contains(t, resp.Results[0].Error, "23: authorization denied",
				"DDL %q should return authorization denied for RW", ddl)
		})
	}
}

func TestAuthorizerTempTableBlockingViaHTTP(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})

	t.Run("rw cannot create temp table", func(t *testing.T) {
		resp := server.Query(t, "read-write-token", harness.Query{
			Query: "CREATE TEMP TABLE tmp_rw_http (id INTEGER PRIMARY KEY)",
		})
		require.Equal(t, "error", resp.Results[0].Type)
		require.Contains(t, resp.Results[0].Error, "23: authorization denied")
	})

	t.Run("ro cannot create temp table", func(t *testing.T) {
		resp := server.Query(t, "read-only-token", harness.Query{
			Query: "CREATE TEMP TABLE tmp_ro_http (id INTEGER PRIMARY KEY)",
		})
		require.Equal(t, "error", resp.Results[0].Type)
		require.Contains(t, resp.Results[0].Error, "23: authorization denied")
	})
}

func TestAuthorizerBatchMixedOperations(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "read-write-token",
		AuthTokenRO: "read-only-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE batch_mixed (id INTEGER PRIMARY KEY, name TEXT);",
	})

	t.Run("rw batch function and dml", func(t *testing.T) {
		resp := server.Query(t, "read-write-token",
			harness.Query{Query: "SELECT COALESCE(NULL, 'x')"},
			harness.Query{Query: "INSERT INTO batch_mixed (name) VALUES ('batch')"},
			harness.Query{Query: "SELECT COUNT(*) FROM batch_mixed"},
		)
		require.Len(t, resp.Results, 3)
		require.Equal(t, "read", resp.Results[0].Type)
		require.Equal(t, "write", resp.Results[1].Type)
		require.Equal(t, "read", resp.Results[2].Type)
		require.Equal(t, [][]any{{float64(1)}}, resp.Results[2].Rows)
	})

	t.Run("ro batch allowed functions plus blocked dml", func(t *testing.T) {
		roResp := server.PostJSON(t, "/query", []harness.Query{
			{Query: "SELECT COALESCE(NULL, 'safe')"},
			{Query: "INSERT INTO batch_mixed (name) VALUES ('blocked')"},
			{Query: "SELECT JSON_VALID('{}')"},
		}, "read-only-token")
		require.Equal(t, http.StatusOK, roResp.StatusCode)
		resp := harness.DecodeJSON[harness.QueryResponse](t, roResp).WithoutTiming()
		require.Len(t, resp.Results, 3)
		require.Equal(t, "read", resp.Results[0].Type)
		require.Equal(t, "error", resp.Results[1].Type)
		require.Contains(t, resp.Results[1].Error, "23: authorization denied")
		require.Equal(t, "read", resp.Results[2].Type)
	})
}

func TestAuthorizerReadOnlyAllowableOperations(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRO: "read-only-token",
	})

	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE ro_allowable (id INTEGER PRIMARY KEY, name TEXT, data JSON);",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "INSERT INTO ro_allowable (name, data) VALUES ('alice', '{\"age\":30}'), ('bob', '{\"age\":25}')",
	})

	t.Run("select with functions", func(t *testing.T) {
		resp := server.Query(t, "read-only-token", harness.Query{
			Query: "SELECT COUNT(*), MIN(name), MAX(name), UPPER(name) FROM ro_allowable",
		})
		require.Equal(t, "read", resp.Results[0].Type)
	})

	t.Run("select with json functions", func(t *testing.T) {
		resp := server.Query(t, "read-only-token", harness.Query{
			Query: "SELECT JSON_EXTRACT(data, '$.age') AS age FROM ro_allowable",
		})
		require.Equal(t, "read", resp.Results[0].Type)
	})

	t.Run("select with pragma", func(t *testing.T) {
		resp := server.Query(t, "read-only-token", harness.Query{
			Query: "PRAGMA table_info(ro_allowable)",
		})
		require.Equal(t, "read", resp.Results[0].Type)
		require.Greater(t, len(resp.Results[0].Rows), 0)
	})

	t.Run("cannot insert even with allowed functions mixed in", func(t *testing.T) {
		resp := server.Query(t, "read-only-token", harness.Query{
			Query: "INSERT INTO ro_allowable (name) VALUES (COALESCE(NULL, 'blocked'))",
		})
		require.Equal(t, "error", resp.Results[0].Type)
		require.Contains(t, resp.Results[0].Error, "23: authorization denied")
	})
}
