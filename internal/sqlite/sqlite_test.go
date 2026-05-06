package sqlite

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSQLiteC(t *testing.T) {
	t.Run("OpenClose", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		assert.NotNil(t, conn)
		assert.NoError(t, conn.Close())
	})

	t.Run("CreateTable", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE test (id INTEGER PRIMARY KEY, val TEXT)", nil)
		assert.NoError(t, err)
	})

	t.Run("InsertMultipleTypes", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query(`
			CREATE TABLE test_types (
				id INTEGER PRIMARY KEY,
				flag BOOLEAN,
				num_int INTEGER,
				num_float REAL,
				txt TEXT,
				bytes BLOB,
				nullable TEXT
			)
		`, nil)
		assert.NoError(t, err)

		res, err := conn.Query(
			`
				INSERT INTO test_types (flag, num_int, num_float, txt, bytes, nullable)
				VALUES (?, ?, ?, ?, ?, ?)
			`,
			[]QueryParam{
				{Value: true},
				{Value: 123},
				{Value: 3.14},
				{Value: "hola"},
				{Value: []byte("raw")},
				{Value: nil},
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), res.RowsAffected)

		selRes, err := conn.Query(
			"SELECT flag, num_int, num_float, txt, bytes, nullable FROM test_types",
			nil,
		)
		assert.NoError(t, err)
		assert.Len(t, selRes.Rows, 1)
		row := selRes.Rows[0]

		assert.Equal(t, 1, row[0])
		assert.Equal(t, 123, row[1])
		assert.Equal(t, 3.14, row[2])
		assert.Equal(t, "hola", row[3])
		assert.Equal(t, []byte("raw"), row[4])
		assert.Nil(t, row[5])
	})

	t.Run("InsertNamedParameter", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE named_test (id INTEGER PRIMARY KEY, value TEXT)", nil)
		assert.NoError(t, err)

		runTest := func(nameForQuery, nameForParam string) {
			value := uuid.NewString()

			_, err = conn.Query(
				fmt.Sprintf("INSERT INTO named_test (value) VALUES (%s)", nameForQuery),
				[]QueryParam{
					{Name: nameForParam, Value: value},
				},
			)
			assert.NoError(t, err)

			res, err := conn.Query(
				"SELECT value FROM named_test ORDER BY id DESC LIMIT 1",
				nil,
			)
			assert.NoError(t, err)
			assert.Len(t, res.Rows, 1)
			assert.Equal(t, value, res.Rows[0][0])
		}

		// Support for all the variants: https://www.sqlite.org/lang_expr.html#varparam
		runTest("?123", "?123")
		runTest("?1", "")
		runTest("?", "")
		runTest(":val", ":val")
		runTest(":val", "val")
		runTest("@val", "@val")
		runTest("@val", "val")
		runTest("$val", "$val")
		runTest("$val", "val")
		runTest("$val::test", "$val::test")
		runTest("$val::test", "val::test")
		runTest("$val(test)", "$val(test)")
		runTest("$val(test)", "val(test)")
	})

	t.Run("MultipleRows", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE multi (id INTEGER PRIMARY KEY, val TEXT)", nil)
		assert.NoError(t, err)

		for i := 1; i <= 3; i++ {
			params := []QueryParam{{Value: i}}
			_, err = conn.Query("INSERT INTO multi (val) VALUES (?)", params)
			assert.NoError(t, err)
		}

		res, err := conn.Query("SELECT id, val FROM multi", nil)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(res.Rows))
	})

	t.Run("UpdateAndRowsAffected", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE upd (id INTEGER PRIMARY KEY, val TEXT)", nil)
		assert.NoError(t, err)
		_, err = conn.Query("INSERT INTO upd (val) VALUES ('original')", nil)
		assert.NoError(t, err)

		res, err := conn.Query("UPDATE upd SET val='nuevo' WHERE id=1", nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), res.RowsAffected)
	})

	t.Run("DeleteAndRowsAffectedZero", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE del (id INTEGER PRIMARY KEY, val TEXT)", nil)
		assert.NoError(t, err)
		_, err = conn.Query("INSERT INTO del (val) VALUES ('abc')", nil)
		assert.NoError(t, err)

		res, err := conn.Query("DELETE FROM del WHERE id=999", nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), res.RowsAffected)
	})

	t.Run("StepNoColumnCount", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		res, err := conn.Query("CREATE TABLE step_test (id INTEGER PRIMARY KEY)", nil)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 0, len(res.Columns))
	})

	t.Run("ReadOnlyCheck", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE test (id INTEGER PRIMARY KEY, val TEXT)", nil)
		assert.NoError(t, err)

		stmt, err := conn.Prepare("INSERT INTO test (val) VALUES (?)")
		assert.NoError(t, err)
		assert.False(t, stmt.ReadOnly())
		assert.NoError(t, stmt.Finalize())

		stmt, err = conn.Prepare("SELECT * FROM test")
		assert.NoError(t, err)
		assert.True(t, stmt.ReadOnly())
		assert.NoError(t, stmt.Finalize())
	})

	t.Run("FinalizeError", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Simulate a nil stmt to check that it doesn't crash
		stmt := &Stmt{}
		err = stmt.Finalize()
		assert.NoError(t, err)
	})

	t.Run("LargeBlob", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		_, err = conn.Query("CREATE TABLE blobtest (id INTEGER PRIMARY KEY, data BLOB)", nil)
		assert.NoError(t, err)

		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		params := []QueryParam{{Value: largeData}}
		_, err = conn.Query("INSERT INTO blobtest(data) VALUES(?)", params)
		assert.NoError(t, err)

		sel, err := conn.Query("SELECT data FROM blobtest", nil)
		assert.NoError(t, err)
		assert.Len(t, sel.Rows, 1)
		assert.Equal(t, largeData, sel.Rows[0][0])
	})

	t.Run("Transactions", func(t *testing.T) {
		conn, err := Open(":memory:")
		assert.NoError(t, err)
		defer func() { _ = conn.Close() }()

		recreateTable := func() {
			_, err = conn.Query("DROP TABLE IF EXISTS test", nil)
			assert.NoError(t, err)
			_, err = conn.Query("CREATE TABLE test (id INTEGER PRIMARY KEY, val TEXT)", nil)
			assert.NoError(t, err)
		}

		t.Run("Successful", func(t *testing.T) {
			recreateTable()

			_, err := conn.Query("BEGIN TRANSACTION", nil)
			assert.NoError(t, err)

			for range 20 {
				_, err = conn.Query(
					"INSERT INTO test (val) VALUES (?)",
					[]QueryParam{{Value: uuid.NewString()}},
				)
				assert.NoError(t, err)
			}

			_, err = conn.Query("COMMIT", nil)
			assert.NoError(t, err)

			sel, err := conn.Query("SELECT val FROM test", nil)
			assert.NoError(t, err)
			assert.Len(t, sel.Rows, 20)
		})

		t.Run("Rollback", func(t *testing.T) {
			recreateTable()

			_, err := conn.Query("BEGIN TRANSACTION", nil)
			assert.NoError(t, err)

			for range 20 {
				_, err = conn.Query(
					"INSERT INTO test (val) VALUES (?)",
					[]QueryParam{{Value: uuid.NewString()}},
				)
				assert.NoError(t, err)
			}

			_, err = conn.Query("ROLLBACK", nil)
			assert.NoError(t, err)

			sel, err := conn.Query("SELECT val FROM test", nil)
			assert.NoError(t, err)
			assert.Len(t, sel.Rows, 0)
		})
	})
}
