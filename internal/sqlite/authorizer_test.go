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
