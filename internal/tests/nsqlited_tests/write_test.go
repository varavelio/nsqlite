package nsqlited_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	t.Run("Basic write", func(t *testing.T) {
		url := createServer(t) + "/query"

		res := sendQuery(t, url, Query{
			Query: "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT);",
		})
		assert.Equal(t, res.Results[0].Type, "write")

		res = sendQuery(t, url, Query{
			Query: "INSERT INTO test (name) VALUES ('test');",
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].RowsAffected, int64(1))
		assert.Equal(t, res.Results[0].LastInsertID, int64(1))
	})

	// TODO: Test concurrent writes
}
