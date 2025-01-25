package nsqlited_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	t.Run("Test sqlite types", func(t *testing.T) {
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
			t.Run("Test sqlite type: "+sqliteType, func(t *testing.T) {
				url := createServer(t) + "/query"

				assertQuery(
					t, url,
					Query{
						Query: "CREATE TABLE test_table (id INTEGER PRIMARY KEY, test_column " + sqliteType + ");",
					},
					Response{
						Results: []ResponseResult{
							{
								Type: "write",
							},
						},
					},
				)

				assertQuery(
					t, url,
					Query{
						Query: "SELECT * FROM test_table;",
					},
					Response{
						Results: []ResponseResult{
							{
								Type:    "read",
								Columns: []string{"id", "test_column"},
								Types:   []string{"INTEGER", sqliteType},
							},
						},
					},
				)
			})
		}
	})

	t.Run("Test type inference", func(t *testing.T) {
		t.Run("Select raw values", func(t *testing.T) {
			url := createServer(t) + "/query"

			assertQuery(
				t, url,
				Query{
					Query: `
						SELECT
							null,
							1,
							1.0,
							'test',
							true,
							'2025-01-01',
							'2025-01-01 12:00:00',
							x'48656C6C6F'
						;`,
				},
				Response{
					Results: []ResponseResult{
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
				},
			)
		})

		t.Run("Select altered values", func(t *testing.T) {
			url := createServer(t) + "/query"

			assertQuery(
				t, url,
				Query{
					Query: `
						CREATE TABLE test_table (
							id INTEGER PRIMARY KEY,
							test_integer INTEGER,
							test_real REAL,
							test_text TEXT
						);
					`,
				},
				Response{
					Results: []ResponseResult{
						{
							Type: "write",
						},
					},
				},
			)

			assertQuery(
				t, url,
				Query{
					Query: `
						INSERT INTO test_table (
							test_integer,
							test_real,
							test_text
						) VALUES (
							1,
							1.0,
							'test'
						);
					`,
				},
				Response{
					Results: []ResponseResult{
						{
							Type:         "write",
							RowsAffected: 1,
							LastInsertID: 1,
						},
					},
				},
			)

			assertQuery(
				t, url,
				Query{
					Query: `
						SELECT
							test_integer + 1,
							test_real + 1,
							test_text || 'test'
						FROM test_table;
					`,
				},
				Response{
					Results: []ResponseResult{
						{
							Type: "read",
							Columns: []string{
								"test_integer + 1",
								"test_real + 1",
								"test_text || 'test'",
							},
							Types: []string{
								"INTEGER",
								"REAL",
								"TEXT",
							},
							Rows: [][]any{
								{
									float64(2),
									float64(2.0),
									"testtest",
								},
							},
						},
					},
				},
			)
		})

		t.Run("Date and time functions", func(t *testing.T) {
			url := createServer(t) + "/query"

			resp := sendQuery(t, url, Query{
				Query: `
					SELECT
						date(),
						time(),
						datetime(),
						julianday(),
						unixepoch(),
						strftime('%Y-%m-%d %H:%M:%S', datetime()),
						timediff(datetime(), datetime())
					;
				`,
			})

			assert.Equal(t, resp.Results[0].Columns, []string{
				"date()",
				"time()",
				"datetime()",
				"julianday()",
				"unixepoch()",
				"strftime('%Y-%m-%d %H:%M:%S', datetime())",
				"timediff(datetime(), datetime())",
			})

			assert.Equal(t, resp.Results[0].Types, []string{
				"TEXT",
				"TEXT",
				"TEXT",
				"REAL",
				"INTEGER",
				"TEXT",
				"TEXT",
			})
		})
	})
}
