package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/logger"
	"github.com/varavelio/nsqlite/internal/stats"
)

func TestRQLiteCompatibility(t *testing.T) {
	t.Run("executes JSON writes and GET queries with rqlite response shape", func(t *testing.T) {
		server := newRQLiteTestServer(t)

		executeResponse := doRQLiteRequest(
			t,
			server,
			http.MethodPost,
			"/db/execute?timings",
			`[
				"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)",
				["INSERT INTO users(name, age) VALUES(?, ?)", "fiona", 20]
			]`,
			"admin-token",
		)
		require.Equal(t, http.StatusOK, executeResponse.Code)

		var executeBody rqliteTestResponse
		decodeRQLiteResponse(t, executeResponse, &executeBody)
		executeResults := rqliteResults(t, executeBody)
		require.Len(t, executeResults, 2)
		require.Equal(t, float64(1), executeResults[1]["last_insert_id"])
		require.Equal(t, float64(1), executeResults[1]["rows_affected"])
		require.Contains(t, executeBody, "time")

		queryResponse := doRQLiteRequest(
			t,
			server,
			http.MethodGet,
			"/db/query?timings&q=SELECT%20id%2C%20name%2C%20age%20FROM%20users",
			"",
			"ro-token",
		)
		require.Equal(t, http.StatusOK, queryResponse.Code)

		var queryBody rqliteTestResponse
		decodeRQLiteResponse(t, queryResponse, &queryBody)
		queryResults := rqliteResults(t, queryBody)
		require.Len(t, queryResults, 1)
		require.Equal(t, []any{"id", "name", "age"}, queryResults[0]["columns"])
		require.Equal(t, []any{"integer", "text", "integer"}, queryResults[0]["types"])
		require.Equal(
			t,
			[]any{float64(1), "fiona", float64(20)},
			queryResults[0]["values"].([]any)[0],
		)
	})

	t.Run("uses the Basic auth password as the NSQLite token", func(t *testing.T) {
		server := newRQLiteTestServer(t)

		response := doRQLiteRequest(
			t,
			server,
			http.MethodPost,
			"/db/query",
			`["SELECT 1"]`,
			"wrong-token",
		)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run(
		"supports associative rows and named parameters through the unified endpoint",
		func(t *testing.T) {
			server := newRQLiteTestServer(t)

			setupResponse := doRQLiteRequest(
				t,
				server,
				http.MethodPost,
				"/db/execute",
				`["CREATE TABLE people (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"]`,
				"admin-token",
			)
			require.Equal(t, http.StatusOK, setupResponse.Code)

			response := doRQLiteRequest(
				t,
				server,
				http.MethodPost,
				"/db/request?associative&timings",
				`[
				["INSERT INTO people(name, age) VALUES(:name, :age)", {"name": "declan", "age": 30}],
				["SELECT name, age FROM people WHERE name=:name", {"name": "declan"}]
			]`,
				"rw-token",
			)
			require.Equal(t, http.StatusOK, response.Code)

			var body rqliteTestResponse
			decodeRQLiteResponse(t, response, &body)
			results := rqliteResults(t, body)
			require.Len(t, results, 2)
			require.Equal(t, float64(1), results[0]["rows_affected"])
			require.Equal(t, map[string]any{"age": "integer", "name": "text"}, results[1]["types"])
			require.Equal(
				t,
				map[string]any{"age": float64(30), "name": "declan"},
				results[1]["rows"].([]any)[0],
			)
		},
	)

	t.Run("rolls back transaction requests when one statement fails", func(t *testing.T) {
		server := newRQLiteTestServer(t)

		setupResponse := doRQLiteRequest(
			t,
			server,
			http.MethodPost,
			"/db/execute",
			`["CREATE TABLE rollback_items (id INTEGER PRIMARY KEY, name TEXT)"]`,
			"admin-token",
		)
		require.Equal(t, http.StatusOK, setupResponse.Code)

		response := doRQLiteRequest(
			t,
			server,
			http.MethodPost,
			"/db/execute?transaction",
			`[
				["INSERT INTO rollback_items(id, name) VALUES(?, ?)", 1, "first"],
				["INSERT INTO rollback_items(id, name) VALUES(?, ?)", 1, "duplicate"]
			]`,
			"rw-token",
		)
		require.Equal(t, http.StatusOK, response.Code)

		var body rqliteTestResponse
		decodeRQLiteResponse(t, response, &body)
		results := rqliteResults(t, body)
		require.Len(t, results, 2)
		require.Contains(t, results[1]["error"], "constraint")

		queryResponse := doRQLiteRequest(
			t,
			server,
			http.MethodGet,
			"/db/query?q=SELECT%20COUNT(*)%20FROM%20rollback_items",
			"",
			"ro-token",
		)
		require.Equal(t, http.StatusOK, queryResponse.Code)

		var queryBody rqliteTestResponse
		decodeRQLiteResponse(t, queryResponse, &queryBody)
		queryResults := rqliteResults(t, queryBody)
		require.Equal(t, float64(0), queryResults[0]["values"].([]any)[0].([]any)[0])
	})
}

type rqliteTestResponse map[string]any

func rqliteResults(t *testing.T, body rqliteTestResponse) []map[string]any {
	t.Helper()
	rawResults, ok := body["results"].([]any)
	require.True(t, ok, "results is missing or has the wrong type")

	results := make([]map[string]any, 0, len(rawResults))
	for _, rawResult := range rawResults {
		result, ok := rawResult.(map[string]any)
		require.True(t, ok, "result has the wrong type")
		results = append(results, result)
	}

	return results
}

func newRQLiteTestServer(t *testing.T) *Server {
	t.Helper()

	dbStats := stats.NewDBStats()
	t.Cleanup(dbStats.Close)

	dbInstance, err := db.NewDB(db.Config{
		Logger:        logger.NewLogger(),
		DBStats:       dbStats,
		DataDir:       t.TempDir(),
		TxIdleTimeout: 10 * time.Second,
		MaxReadConns:  2,
		CacheSizeKB:   20000,
		BusyTimeout:   5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, dbInstance.Close()) })

	server, err := NewServer(Config{
		Logger:              logger.NewLogger(),
		DBStats:             dbStats,
		DB:                  dbInstance,
		AuthTokens:          []string{"admin-token"},
		ReadWriteAuthTokens: []string{"rw-token"},
		ReadOnlyAuthTokens:  []string{"ro-token"},
	})
	require.NoError(t, err)

	return server
}

func doRQLiteRequest(
	t *testing.T,
	server *Server,
	method string,
	path string,
	body string,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Reader
	if body == "" {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = bytes.NewReader([]byte(body))
	}

	request := httptest.NewRequestWithContext(context.Background(), method, path, requestBody)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		credentials := base64.StdEncoding.EncodeToString([]byte("ignored-user:" + token))
		request.Header.Set("Authorization", "Basic "+credentials)
	}

	response := httptest.NewRecorder()
	server.createMux().ServeHTTP(response, request)
	return response
}

func decodeRQLiteResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.NoError(
		t,
		json.Unmarshal(response.Body.Bytes(), target),
		strings.TrimSpace(response.Body.String()),
	)
}
