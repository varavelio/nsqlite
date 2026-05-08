package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthorizerRoleEnforcement(t *testing.T) {
	t.Run("read write denies ddl but allows dml", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))

		_, err = conn.Query("ALTER TABLE widgets ADD COLUMN description TEXT", nil)
		require.ErrorContains(t, err, "23: authorization denied")

		insert, err := conn.Query("INSERT INTO widgets (name) VALUES ('gizmo')", nil)
		require.NoError(t, err)
		require.Equal(t, int64(1), insert.RowsAffected)

		update, err := conn.Query("UPDATE widgets SET name = 'gizmo-2' WHERE id = 1", nil)
		require.NoError(t, err)
		require.Equal(t, int64(1), update.RowsAffected)
	})

	t.Run("read only denies writes and transactions", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE audit_log (id INTEGER PRIMARY KEY, message TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))

		_, err = conn.Query("BEGIN TRANSACTION", nil)
		require.ErrorContains(t, err, "23: authorization denied")

		_, err = conn.Query("INSERT INTO audit_log (message) VALUES ('blocked')", nil)
		require.ErrorContains(t, err, "23: authorization denied")

		read, err := conn.Query("SELECT COUNT(*) FROM audit_log", nil)
		require.NoError(t, err)
		require.Equal(t, [][]any{{0}}, read.Rows)
	})

	t.Run("role changes apply to the same connection", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE role_switch (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("DROP TABLE role_switch", nil)
		require.ErrorContains(t, err, "23: authorization denied")

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("ALTER TABLE role_switch ADD COLUMN created_at TEXT", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
		_, err = conn.Query("INSERT INTO role_switch (name) VALUES ('blocked')", nil)
		require.ErrorContains(t, err, "23: authorization denied")

		read, err := conn.Query("SELECT COUNT(*) FROM role_switch", nil)
		require.NoError(t, err)
		require.Equal(t, [][]any{{0}}, read.Rows)
	})
}

func TestAuthorizerFunctionWhitelist(t *testing.T) {
	setupConn := func(t *testing.T) *Conn {
		t.Helper()
		conn, err := Open(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}

	whitelistedScalars := []struct {
		name string
		sql  string
	}{
		{name: "coalesce", sql: "SELECT COALESCE(NULL, 'default')"},
		{name: "lower/upper", sql: "SELECT LOWER(UPPER('Hello'))"},
		{name: "substr", sql: "SELECT SUBSTR('hello', 2, 3)"},
		{name: "length", sql: "SELECT LENGTH('abc')"},
		{name: "trim/ltrim/rtrim", sql: "SELECT TRIM('  x  '), LTRIM(' a'), RTRIM('b ')"},
		{name: "replace", sql: "SELECT REPLACE('abc', 'b', 'x')"},
		{name: "instr", sql: "SELECT INSTR('hello', 'll')"},
		{name: "abs", sql: "SELECT ABS(-42)"},
		{name: "round", sql: "SELECT ROUND(3.14159, 2)"},
		{name: "hex/unhex", sql: "SELECT HEX('x'), UNHEX('FF')"},
		{name: "typeof", sql: "SELECT TYPEOF(42)"},
		{name: "iif", sql: "SELECT IIF(1 > 0, 'yes', 'no')"},
		{name: "nullif", sql: "SELECT NULLIF(1, 1)"},
		{name: "ifnull", sql: "SELECT IFNULL(NULL, 'fallback')"},
		{name: "quote", sql: "SELECT QUOTE('hello')"},
		{name: "printf", sql: "SELECT PRINTF('val=%d', 42)"},
		{name: "sign", sql: "SELECT SIGN(-10)"},
		{name: "random", sql: "SELECT RANDOM()"},
		{name: "randomblob", sql: "SELECT LENGTH(RANDOMBLOB(8))"},
		{name: "likely/unlikely", sql: "SELECT LIKELY(1), UNLIKELY(0)"},
		{name: "unicode", sql: "SELECT UNICODE('A')"},
		{name: "char", sql: "SELECT CHAR(65)"},
		{name: "octet_length", sql: "SELECT OCTET_LENGTH('abc')"},
		{name: "likelihood", sql: "SELECT LIKELIHOOD(1, 0.5)"},
		{name: "sqlite_version", sql: "SELECT SQLITE_VERSION()"},
		{name: "sqlite_source_id", sql: "SELECT SQLITE_SOURCE_ID()"},
		{name: "total_changes", sql: "SELECT TOTAL_CHANGES()"},
		{name: "last_insert_rowid", sql: "SELECT LAST_INSERT_ROWID()"},
		{name: "zeroblob", sql: "SELECT LENGTH(ZEROBLOB(16))"},
		{
			name: "date functions",
			sql:  "SELECT DATE('now'), TIME('now'), DATETIME('now'), STRFTIME('%Y', 'now'), JULIANDAY('now')",
		},
		{name: "timediff", sql: "SELECT TIMEDIFF('2024-01-01', '2023-01-01')"},
	}

	for _, tc := range whitelistedScalars {
		t.Run("rw_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
			_, err := conn.Query(tc.sql, nil)
			require.NoError(t, err, "function %q should be allowed for RW", tc.name)
		})
		t.Run("ro_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
			_, err := conn.Query(tc.sql, nil)
			require.NoError(t, err, "function %q should be allowed for RO", tc.name)
		})
	}

	t.Run("rw load_extension is blocked", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err := conn.Query("SELECT load_extension('nonexistent')", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "load_extension",
			"load_extension must be blocked for RW (blocked by SQLite core)")
	})

	t.Run("ro load_extension is blocked", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
		_, err := conn.Query("SELECT load_extension('nonexistent')", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "load_extension",
			"load_extension must be blocked for RO (blocked by SQLite core)")
	})

	t.Run("admin load_extension is allowed to prepare", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err := conn.Query("SELECT load_extension('nonexistent')", nil)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "authorization denied")
	})

	t.Run("rw_aggregate_functions", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err := conn.Query("CREATE TABLE agg_test (grp INTEGER, val INTEGER)", nil)
		require.NoError(t, err)
		_, err = conn.Query("INSERT INTO agg_test VALUES (1, 10), (1, 20), (2, 5)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query(
			"SELECT grp, COUNT(*), SUM(val), AVG(val), MIN(val), MAX(val), TOTAL(val), GROUP_CONCAT(val) FROM agg_test GROUP BY grp",
			nil,
		)
		require.NoError(t, err)
	})

	t.Run("ro_json_functions", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
		_, err := conn.Query(
			"SELECT JSON('{\"a\":1}'), JSON_EXTRACT('{\"a\":1}', '$.a'), JSON_TYPE('{\"a\":1}'), JSON_VALID('{\"a\":1}'), JSON_ARRAY(1,2,3), JSON_OBJECT('k','v'), JSON_ARRAY_LENGTH('[1,2,3]'), JSON_QUOTE('hello')",
			nil,
		)
		require.NoError(t, err)
	})

	t.Run("rw_json_mutating_functions_are_safe", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err := conn.Query(
			"SELECT JSON_INSERT('{}', '$.a', 1), JSON_REPLACE('{\"a\":1}', '$.a', 2), JSON_SET('{}', '$.a', 1), JSON_REMOVE('{\"a\":1}', '$.a'), JSON_PATCH('{}', '{\"a\":1}')",
			nil,
		)
		require.NoError(t, err)
	})
}

func TestAuthorizerPragmaWhitelist(t *testing.T) {
	setupConn := func(t *testing.T) *Conn {
		t.Helper()
		conn, err := Open(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}

	allowedPragmas := []struct {
		name string
		sql  string
	}{
		{name: "table_info", sql: "PRAGMA table_info(sqlite_master)"},
		{name: "table_xinfo", sql: "PRAGMA table_xinfo(sqlite_master)"},
		{name: "index_list", sql: "PRAGMA index_list(sqlite_master)"},
		{name: "index_info", sql: "PRAGMA index_info(sqlite_autoindex_sqlite_master_1)"},
		{name: "index_xinfo", sql: "PRAGMA index_xinfo(sqlite_autoindex_sqlite_master_1)"},
		{name: "foreign_key_list", sql: "PRAGMA foreign_key_list(sqlite_master)"},
		{name: "database_list", sql: "PRAGMA database_list"},
		{name: "compile_options", sql: "PRAGMA compile_options"},
		{name: "schema_version", sql: "PRAGMA schema_version"},
		{name: "user_version", sql: "PRAGMA user_version"},
		{name: "collation_list", sql: "PRAGMA collation_list"},
		{name: "function_list", sql: "PRAGMA function_list"},
		{name: "module_list", sql: "PRAGMA module_list"},
		{name: "pragma_list", sql: "PRAGMA pragma_list"},
		{name: "page_count", sql: "PRAGMA page_count"},
		{name: "page_size", sql: "PRAGMA page_size"},
		{name: "freelist_count", sql: "PRAGMA freelist_count"},
		{name: "data_version", sql: "PRAGMA data_version"},
	}

	for _, tc := range allowedPragmas {
		t.Run("rw_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
			_, err := conn.Query(tc.sql, nil)
			require.NoError(t, err, "pragma %q should be allowed for RW", tc.name)
		})
		t.Run("ro_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
			_, err := conn.Query(tc.sql, nil)
			require.NoError(t, err, "pragma %q should be allowed for RO", tc.name)
		})
	}

	blockedPragmas := []struct {
		name string
		sql  string
	}{
		{name: "journal_mode", sql: "PRAGMA journal_mode = WAL"},
		{name: "synchronous", sql: "PRAGMA synchronous = NORMAL"},
		{name: "foreign_keys", sql: "PRAGMA foreign_keys = ON"},
		{name: "cache_size", sql: "PRAGMA cache_size = -2000"},
		{name: "mmap_size", sql: "PRAGMA mmap_size = 0"},
		{name: "temp_store", sql: "PRAGMA temp_store = MEMORY"},
		{name: "busy_timeout", sql: "PRAGMA busy_timeout = 5000"},
		{name: "writable_schema", sql: "PRAGMA writable_schema = ON"},
		{name: "wal_checkpoint", sql: "PRAGMA wal_checkpoint"},
		{name: "auto_vacuum", sql: "PRAGMA auto_vacuum"},
		{name: "analysis_limit", sql: "PRAGMA analysis_limit = 1000"},
		{name: "application_id", sql: "PRAGMA application_id"},
		{name: "encoding", sql: "PRAGMA encoding"},
	}

	for _, tc := range blockedPragmas {
		t.Run("rw_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
			_, err := conn.Query(tc.sql, nil)
			require.ErrorContains(
				t,
				err,
				"23: authorization denied",
				"pragma %q should be blocked for RW",
				tc.name,
			)
		})
		t.Run("ro_"+tc.name, func(t *testing.T) {
			conn := setupConn(t)
			require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
			_, err := conn.Query(tc.sql, nil)
			require.ErrorContains(
				t,
				err,
				"23: authorization denied",
				"pragma %q should be blocked for RO",
				tc.name,
			)
		})
	}

	t.Run("admin writable pragma is allowed", func(t *testing.T) {
		conn := setupConn(t)
		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err := conn.Query("PRAGMA journal_mode = MEMORY", nil)
		require.NoError(t, err)
	})
}

func TestAuthorizerTriggerAgainstInternalTable(t *testing.T) {
	t.Run("rw cannot insert into sqlite_master", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("INSERT INTO sqlite_master VALUES ('table', 'x', 'x', 0, '')", nil)
		require.ErrorContains(t, err, "23: authorization denied",
			"the authorizer must deny INSERT into sqlite_master before SQLite's own checks")
	})
}

func TestAuthorizerTempTableBlocking(t *testing.T) {
	t.Run("rw cannot create temp table", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("CREATE TEMP TABLE tmp_rw (id INTEGER PRIMARY KEY)", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("ro cannot create temp table", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
		_, err = conn.Query("CREATE TEMP TABLE tmp_ro (id INTEGER PRIMARY KEY)", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot create temp view", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("CREATE TEMP VIEW tmp_vw AS SELECT 1", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot create temp trigger", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE temp_target (id INTEGER PRIMARY KEY)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query(
			"CREATE TEMP TRIGGER tmp_trg AFTER INSERT ON temp_target BEGIN SELECT 1; END",
			nil,
		)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("admin can create temp table", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TEMP TABLE admin_tmp (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)
		_, err = conn.Query("INSERT INTO admin_tmp (name) VALUES ('admin')", nil)
		require.NoError(t, err)
		res, err := conn.Query("SELECT COUNT(*) FROM admin_tmp", nil)
		require.NoError(t, err)
		require.Equal(t, [][]any{{1}}, res.Rows)
	})
}

func TestAuthorizerEdgeCases(t *testing.T) {
	t.Run("rw cannot ATTACH database", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("ATTACH DATABASE ':memory:' AS aux", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot DETACH database", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("ATTACH DATABASE ':memory:' AS aux2", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("DETACH DATABASE aux2", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot REINDEX", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE reidx_test (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)
		_, err = conn.Query("CREATE INDEX idx_reidx_name ON reidx_test(name)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("REINDEX idx_reidx_name", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot CREATE VIEW", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE view_base (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("CREATE VIEW rw_view AS SELECT id, name FROM view_base", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot DROP VIEW", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE vd_base (id INTEGER PRIMARY KEY)", nil)
		require.NoError(t, err)
		_, err = conn.Query("CREATE VIEW drop_view AS SELECT id FROM vd_base", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("DROP VIEW drop_view", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot CREATE TRIGGER", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE trig_base (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query(
			"CREATE TRIGGER rw_trg AFTER INSERT ON trig_base BEGIN SELECT 1; END",
			nil,
		)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot DROP TRIGGER", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE dt_base (id INTEGER PRIMARY KEY)", nil)
		require.NoError(t, err)
		_, err = conn.Query(
			"CREATE TRIGGER dt_trg AFTER INSERT ON dt_base BEGIN SELECT 1; END",
			nil,
		)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("DROP TRIGGER dt_trg", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot ANALYZE", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE analyze_test (id INTEGER PRIMARY KEY, val INTEGER)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("ANALYZE analyze_test", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot create virtual table", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("CREATE VIRTUAL TABLE vt_rw USING fts5(content)", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot CREATE INDEX", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE ci_test (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("CREATE INDEX ci_idx ON ci_test(name)", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw cannot DROP INDEX", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE di_test (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)
		_, err = conn.Query("CREATE INDEX di_idx ON di_test(name)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("DROP INDEX di_idx", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("ro cannot use SAVEPOINT", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE sp_test (id INTEGER PRIMARY KEY)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadOnly))
		_, err = conn.Query("SAVEPOINT ro_sp", nil)
		require.ErrorContains(t, err, "23: authorization denied")
	})

	t.Run("rw can use SAVEPOINT", func(t *testing.T) {
		conn, err := Open(":memory:")
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleAdmin))
		_, err = conn.Query("CREATE TABLE sp_rw_test (id INTEGER PRIMARY KEY, name TEXT)", nil)
		require.NoError(t, err)

		require.NoError(t, conn.SetAuthorizerRole(AuthorizerRoleReadWrite))
		_, err = conn.Query("SAVEPOINT rw_sp", nil)
		require.NoError(t, err)
		_, err = conn.Query("INSERT INTO sp_rw_test (name) VALUES ('savepoint')", nil)
		require.NoError(t, err)
		_, err = conn.Query("RELEASE rw_sp", nil)
		require.NoError(t, err)
		res, err := conn.Query("SELECT COUNT(*) FROM sp_rw_test", nil)
		require.NoError(t, err)
		require.Equal(t, [][]any{{1}}, res.Rows)
	})
}
