package query_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestQueryEndpointSupportsParameterBindingStyles(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(t, "", harness.Query{
		Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT, age INTEGER);",
	})

	testCases := []struct {
		name   string
		query  string
		params []harness.QueryParam
		expect []any
	}{
		{
			name:   "no params",
			query:  "INSERT INTO users (name, surname, age) VALUES ('John', 'Doe', 20) RETURNING name, surname, age;",
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:   "nameless params",
			query:  "INSERT INTO users (name, surname, age) VALUES (?, ?, ?) RETURNING name, surname, age;",
			params: []harness.QueryParam{{Value: "John"}, {Value: "Doe"}, {Value: 20}},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:  "indexed params",
			query: "INSERT INTO users (name, surname, age) VALUES (?111, ?222, ?333) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Name: "?111", Value: "John"},
				{Name: "?222", Value: "Doe"},
				{Name: "?333", Value: 20},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:   "ordered indexed params",
			query:  "INSERT INTO users (name, surname, age) VALUES (?1, ?1, ?2) RETURNING name, surname, age;",
			params: []harness.QueryParam{{Value: "John"}, {Value: 20}},
			expect: []any{"John", "John", float64(20)},
		},
		{
			name:  "colon params unordered without prefix",
			query: "INSERT INTO users (name, surname, age) VALUES (:name, :surname, :age) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Name: "age", Value: 20},
				{Name: "surname", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:  "at params unordered without prefix",
			query: "INSERT INTO users (name, surname, age) VALUES (@name, @surname, @age) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Name: "age", Value: 20},
				{Name: "surname", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:  "dollar params unordered without prefix",
			query: "INSERT INTO users (name, surname, age) VALUES ($name, $surname::suffix, $age(suffix)) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Name: "age(suffix)", Value: 20},
				{Name: "surname::suffix", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:  "mixed named prefixes",
			query: "INSERT INTO users (name, surname, age) VALUES (:name, @surname, $age(suffix)) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Name: "$age(suffix)", Value: 20},
				{Name: "@surname", Value: "Doe"},
				{Name: ":name", Value: "John"},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
		{
			name:  "mixed nameless and named params",
			query: "INSERT INTO users (name, surname, age) VALUES (?, @surname, $age(suffix)) RETURNING name, surname, age;",
			params: []harness.QueryParam{
				{Value: "John"},
				{Name: "@surname", Value: "Doe"},
				{Name: "$age(suffix)", Value: 20},
			},
			expect: []any{"John", "Doe", float64(20)},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := server.Query(
				t,
				"",
				harness.Query{Query: testCase.query, Params: testCase.params},
			)
			require.Equal(t, harness.QueryResponse{
				Results: []harness.QueryResult{{
					Type:    "write",
					Columns: []string{"name", "surname", "age"},
					Types:   []string{"TEXT", "TEXT", "INTEGER"},
					Rows:    [][]any{testCase.expect},
				}},
			}, response)
		})
	}
}

func TestQueryEndpointReturnsHelpfulErrorForEmptyQuery(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Query(t, "", harness.Query{Query: ""})

	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:  "error",
			Error: "Empty query",
		}},
	}, response)
}

func TestQueryEndpointReturnsEmptyResultsForAnEmptyBatch(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Query(t, "")

	require.Empty(t, response.Results)
}

func TestQueryEndpointSupportsJSONPrimitiveParameterValues(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Query(t, "", harness.Query{
		Query: "SELECT ?, ?, ?, ?, ?;",
		Params: []harness.QueryParam{
			{Value: nil},
			{Value: true},
			{Value: false},
			{Value: 123},
			{Value: 4.5},
		},
	})

	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"?", "?", "?", "?", "?"},
			Types:   []string{"NULL", "INTEGER", "INTEGER", "INTEGER", "REAL"},
			Rows: [][]any{{
				nil,
				float64(1),
				float64(0),
				float64(123),
				4.5,
			}},
		}},
	}, response)
}

func TestQueryEndpointReturnsResultErrorsForUnsupportedParameterTypes(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	testCases := []struct {
		name  string
		param harness.QueryParam
		error string
	}{
		{
			name:  "array parameter",
			param: harness.QueryParam{Value: []any{"Ada"}},
			error: "exactly one SQLite value field must be set",
		},
		{
			name:  "object parameter",
			param: harness.QueryParam{Value: map[string]any{"name": "Ada"}},
			error: "exactly one SQLite value field must be set",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := server.Query(t, "", harness.Query{
				Query:  "SELECT ?;",
				Params: []harness.QueryParam{testCase.param},
			})

			require.Equal(t, "error", response.Results[0].Type)
			require.Contains(t, response.Results[0].Error, testCase.error)
		})
	}
}

func TestQueryEndpointSupportsAllDeclaredSQLiteTypes(t *testing.T) {
	t.Parallel()

	sqliteTypes := []string{
		"INT",
		"INTEGER",
		"TINYINT",
		"SMALLINT",
		"MEDIUMINT",
		"BIGINT",
		"UNSIGNED BIG INT",
		"INT2",
		"INT8",
		"CHARACTER(20)",
		"VARCHAR(255)",
		"VARYING CHARACTER(255)",
		"NCHAR(55)",
		"NATIVE CHARACTER(70)",
		"NVARCHAR(100)",
		"TEXT",
		"CLOB",
		"BLOB",
		"REAL",
		"DOUBLE",
		"FLOAT",
		"NUMERIC",
		"DECIMAL(10,5)",
		"BOOLEAN",
		"DATE",
		"DATETIME",
	}

	for _, sqliteType := range sqliteTypes {
		t.Run(sqliteType, func(t *testing.T) {
			server := harness.StartServer(t, harness.ServerConfig{})
			server.Query(
				t,
				"",
				harness.Query{
					Query: fmt.Sprintf(
						"CREATE TABLE test_table (id INTEGER PRIMARY KEY, test_column %s);",
						sqliteType,
					),
				},
			)

			response := server.Query(t, "", harness.Query{Query: "SELECT * FROM test_table;"})
			require.Equal(t, harness.QueryResponse{
				Results: []harness.QueryResult{{
					Type:    "read",
					Columns: []string{"id", "test_column"},
					Types:   response.Results[0].Types,
				}},
			}, response)
		})
	}
}

func TestQueryEndpointInfersResultTypes(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	rawValues := server.Query(t, "", harness.Query{Query: `
		SELECT
			null,
			1,
			1.0,
			'test',
			true,
			'2025-01-01',
			'2025-01-01 12:00:00',
			x'48656C6C6F'
		;
	`})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{
			{
				Type: "read",
				Columns: []string{
					"null",
					"1",
					"1.0",
					"'test'",
					"true",
					"'2025-01-01'",
					"'2025-01-01 12:00:00'",
					"x'48656C6C6F'",
				},
				Types: []string{
					"NULL",
					"INTEGER",
					"REAL",
					"TEXT",
					"INTEGER",
					"TEXT",
					"TEXT",
					"BLOB",
				},
				Rows: [][]any{
					{
						nil,
						float64(1),
						1.0,
						"test",
						float64(1),
						"2025-01-01",
						"2025-01-01 12:00:00",
						"SGVsbG8=",
					},
				},
			},
		},
	}, rawValues)

	server.Query(t, "", harness.Query{Query: `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			test_integer INTEGER,
			test_real REAL,
			test_text TEXT
		);
	`})
	server.Query(t, "", harness.Query{Query: `
		INSERT INTO test_table (
			test_integer,
			test_real,
			test_text
		) VALUES (
			1,
			1.0,
			'test'
		);
	`})

	alteredValues := server.Query(t, "", harness.Query{Query: `
		SELECT
			test_integer + 1,
			test_real + 1,
			test_text || 'test'
		FROM test_table;
	`})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"test_integer + 1", "test_real + 1", "test_text || 'test'"},
			Types:   []string{"INTEGER", "REAL", "TEXT"},
			Rows:    [][]any{{float64(2), float64(2), "testtest"}},
		}},
	}, alteredValues)
}

func TestQueryEndpointReportsDateAndTimeFunctionTypes(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Query(t, "", harness.Query{Query: `
		SELECT
			date(),
			time(),
			datetime(),
			julianday(),
			unixepoch(),
			strftime('%Y-%m-%d %H:%M:%S', datetime()),
			timediff(datetime(), datetime())
		;
	`})

	require.Equal(t, []string{
		"date()",
		"time()",
		"datetime()",
		"julianday()",
		"unixepoch()",
		"strftime('%Y-%m-%d %H:%M:%S', datetime())",
		"timediff(datetime(), datetime())",
	}, response.Results[0].Columns)

	require.Equal(
		t,
		[]string{"TEXT", "TEXT", "TEXT", "REAL", "INTEGER", "TEXT", "TEXT"},
		response.Results[0].Types,
	)
}
