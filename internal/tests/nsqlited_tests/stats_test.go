package nsqlited_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {
	t.Run("Empty stats", func(t *testing.T) {
		url := createServer(t)
		stats := getStats(t, url)

		assert.Equal(t, stats.Totals.Reads, int64(1)) // Just the health check read
		assert.Equal(t, stats.Totals.Begins, int64(0))
		assert.Equal(t, stats.Totals.Writes, int64(0))
		assert.Equal(t, stats.Totals.Commits, int64(0))
		assert.Equal(t, stats.Totals.Rollbacks, int64(0))
		assert.Equal(t, stats.Totals.Errors, int64(0))
		assert.Equal(t, stats.Totals.HTTPRequests, int64(0))
		assert.Equal(t, stats.QueuedBegins, int64(0))
		assert.Equal(t, stats.QueuedWrites, int64(0))
		assert.Equal(t, stats.QueuedHTTPRequests, int64(0))
	})

	t.Run("Stats after a read query", func(t *testing.T) {
		url := createServer(t)
		queryURL := url + "/query"

		sendQuery(t, queryURL, Query{
			Query: "SELECT 1;",
		})
		stats := getStats(t, url)

		assert.Equal(t, stats.Totals.Reads, int64(2))
		assert.Equal(t, stats.Totals.Begins, int64(0))
		assert.Equal(t, stats.Totals.Writes, int64(0))
		assert.Equal(t, stats.Totals.Commits, int64(0))
		assert.Equal(t, stats.Totals.Rollbacks, int64(0))
		assert.Equal(t, stats.Totals.Errors, int64(0))
		assert.Equal(t, stats.Totals.HTTPRequests, int64(1))
		assert.Equal(t, stats.QueuedBegins, int64(0))
		assert.Equal(t, stats.QueuedWrites, int64(0))
		assert.Equal(t, stats.QueuedHTTPRequests, int64(0))
	})

	t.Run("Stats after a write query", func(t *testing.T) {
		url := createServer(t)
		queryURL := url + "/query"

		sendQuery(t, queryURL, Query{
			Query: "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT);",
		})

		stats := getStats(t, url)
		assert.Equal(t, stats.Totals.Writes, int64(1))
		assert.Equal(t, stats.Totals.Begins, int64(0))
		assert.Equal(t, stats.Totals.Reads, int64(1))
		assert.Equal(t, stats.Totals.Commits, int64(0))
		assert.Equal(t, stats.Totals.Rollbacks, int64(0))
		assert.Equal(t, stats.Totals.Errors, int64(0))
		assert.Equal(t, stats.Totals.HTTPRequests, int64(1))
		assert.Equal(t, stats.QueuedBegins, int64(0))
		assert.Equal(t, stats.QueuedWrites, int64(0))
		assert.Equal(t, stats.QueuedHTTPRequests, int64(0))
	})

	// TODO: Add more tests to this suite
}
