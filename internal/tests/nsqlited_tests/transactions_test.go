package nsqlited_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactions(t *testing.T) {
	t.Run("Successful transaction", func(t *testing.T) {
		url := createServer(t) + "/query"

		sendQuery(t, url, Query{
			Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT);",
		})

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)
		txID := res.Results[0].TxID

		assertQuery(
			t, url,
			Query{
				Query: "INSERT INTO users (name, surname) VALUES ('John', 'Doe');",
				TxID:  txID,
			},
			Response{
				Results: []ResponseResult{{
					Type:         "write",
					LastInsertID: 1,
					RowsAffected: 1,
				}},
			},
		)

		// Check the count of users inside the transaction
		res = sendQuery(t, url, Query{
			Query: "SELECT COUNT(*) FROM users;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "read")
		assert.Equal(t, res.Results[0].Rows[0][0], float64(1))

		// Check the count of users outside the transaction
		res = sendQuery(t, url, Query{
			Query: "SELECT COUNT(*) FROM users;",
		})
		assert.Equal(t, res.Results[0].Type, "read")
		assert.Equal(t, res.Results[0].Rows[0][0], float64(0))

		res = sendQuery(t, url, Query{
			Query: "COMMIT;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "commit")

		// When the transaction is commited, the count of users should be 1
		res = sendQuery(t, url, Query{
			Query: "SELECT COUNT(*) FROM users;",
		})
		assert.Equal(t, res.Results[0].Type, "read")
		assert.Equal(t, res.Results[0].Rows[0][0], float64(1))
	})

	t.Run("Rollback transaction", func(t *testing.T) {
		url := createServer(t) + "/query"

		sendQuery(t, url, Query{
			Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT);",
		})

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)
		txID := res.Results[0].TxID

		assertQuery(
			t, url,
			Query{
				Query: "INSERT INTO users (name, surname) VALUES ('John', 'Doe');",
				TxID:  txID,
			},
			Response{
				Results: []ResponseResult{{
					Type:         "write",
					LastInsertID: 1,
					RowsAffected: 1,
				}},
			},
		)

		res = sendQuery(t, url, Query{
			Query: "ROLLBACK;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "rollback")

		// When the transaction is rolled back, the count of users should be 0
		res = sendQuery(t, url, Query{
			Query: "SELECT COUNT(*) FROM users;",
		})
		assert.Equal(t, res.Results[0].Type, "read")
		assert.Equal(t, res.Results[0].Rows[0][0], float64(0))
	})

	t.Run("Commit without BEGIN", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "COMMIT;",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Rollback without BEGIN", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "ROLLBACK;",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Commit to non-existent transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "COMMIT;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Rollback to non-existent transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "ROLLBACK;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Commit to incorrect transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)

		res = sendQuery(t, url, Query{
			Query: "COMMIT;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Rollback to incorrect transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)

		res = sendQuery(t, url, Query{
			Query: "ROLLBACK;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Transaction within transaction", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)
		txID := res.Results[0].TxID

		res = sendQuery(t, url, Query{
			Query: "BEGIN;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Query with non-existent transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "SELECT 1, 2, 3;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Query with incorrect transaction ID", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)

		res = sendQuery(t, url, Query{
			Query: "SELECT 1, 2, 3;",
			TxID:  "invalid",
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	t.Run("Query to already committed transaction", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "BEGIN;",
		})
		assert.Equal(t, res.Results[0].Type, "begin")
		assert.NotEmpty(t, res.Results[0].TxID)
		txID := res.Results[0].TxID

		res = sendQuery(t, url, Query{
			Query: "SELECT 1, 2, 3;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "read")

		res = sendQuery(t, url, Query{
			Query: "COMMIT;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "commit")

		res = sendQuery(t, url, Query{
			Query: "SELECT 1, 2, 3;",
			TxID:  txID,
		})
		assert.Equal(t, res.Results[0].Type, "error")
		assert.NotEmpty(t, res.Results[0].Error)
	})

	// TODO: Test concurrent transactions
}
