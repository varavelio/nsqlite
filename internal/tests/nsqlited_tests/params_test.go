package nsqlited_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	url := createServer(t) + "/query"
	sendQuery(t, url, Query{
		Query: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, surname TEXT, age INTEGER);",
	})

	t.Run("No params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES ('John', 'Doe', 20) RETURNING name, surname, age;",
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("Nameless params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (?, ?, ?) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Value: "John"},
				{Value: "Doe"},
				{Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("?NNN named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (?111, ?222, ?333) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "?111", Value: "John"},
				{Name: "?222", Value: "Doe"},
				{Name: "?333", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("?NNN ordered params with ?", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (?1, ?1, ?2) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Value: "John"},
				{Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "John", float64(20)}})
	})

	t.Run(":AAAA named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (:name, :surname, :age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: ":name", Value: "John"},
				{Name: ":surname", Value: "Doe"},
				{Name: ":age", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run(":AAAA named params unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (:name, :surname, :age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: ":age", Value: 20},
				{Name: ":surname", Value: "Doe"},
				{Name: ":name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run(":AAAA named params without prefix", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (:name, :surname, :age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "name", Value: "John"},
				{Name: "surname", Value: "Doe"},
				{Name: "age", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run(":AAAA named params without prefix unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (:name, :surname, :age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "age", Value: 20},
				{Name: "surname", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("@AAAA named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (@name, @surname, @age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "@name", Value: "John"},
				{Name: "@surname", Value: "Doe"},
				{Name: "@age", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("@AAAA named params unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (@name, @surname, @age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "@age", Value: 20},
				{Name: "@surname", Value: "Doe"},
				{Name: "@name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
	})

	t.Run("@AAAA named params without prefix", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (@name, @surname, @age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "name", Value: "John"},
				{Name: "surname", Value: "Doe"},
				{Name: "age", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("@AAAA named params without prefix unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (@name, @surname, @age) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "age", Value: 20},
				{Name: "surname", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("$AAAA named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES ($name, $surname::suffix, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "$name", Value: "John"},
				{Name: "$surname::suffix", Value: "Doe"},
				{Name: "$age(suffix)", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("$AAAA named params unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES ($name, $surname::suffix, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "$age(suffix)", Value: 20},
				{Name: "$surname::suffix", Value: "Doe"},
				{Name: "$name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("$AAAA named params without prefix", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES ($name, $surname::suffix, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "name", Value: "John"},
				{Name: "surname::suffix", Value: "Doe"},
				{Name: "age(suffix)", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("$AAAA named params without prefix unordered", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES ($name, $surname::suffix, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "age(suffix)", Value: 20},
				{Name: "surname::suffix", Value: "Doe"},
				{Name: "name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("Mix all types of named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (:name, @surname, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Name: "$age(suffix)", Value: 20},
				{Name: "@surname", Value: "Doe"},
				{Name: ":name", Value: "John"},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})

	t.Run("Mix nameless and named params", func(t *testing.T) {
		res := sendQuery(t, url, Query{
			Query: "INSERT INTO users (name, surname, age) VALUES (?, @surname, $age(suffix)) RETURNING name, surname, age;",
			Params: []QueryParam{
				{Value: "John"},
				{Name: "@surname", Value: "Doe"},
				{Name: "$age(suffix)", Value: 20},
			},
		})
		assert.Equal(t, res.Results[0].Type, "write")
		assert.Equal(t, res.Results[0].Columns, []string{"name", "surname", "age"})
		assert.Equal(t, res.Results[0].Rows, [][]any{{"John", "Doe", float64(20)}})
	})
}
