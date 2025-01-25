package tests

// We duplicate some types here to check if the types change over time.
// This is a simple way to ensure that the types are not changed without
// noticing.

type (
	// Query represents a single query within a request.
	Query struct {
		TxID   string       `json:"txId"`
		Query  string       `json:"query"`
		Params []QueryParam `json:"params"`
	}

	// QueryParam represents a named (?NNN, :VVV, @VVV, $VVV) or nameless (?) parameter in a SQL query.
	QueryParam struct {
		Name  string `json:"name,omitempty"`
		Value any    `json:"value"`
	}

	// Response represents the structure of an outgoing response.
	Response struct {
		Time    float64          `json:"time"`
		Results []ResponseResult `json:"results"`
	}

	// ResponseResult represents the structure of a query result.
	ResponseResult struct {
		// For all queries
		Type string  `json:"type"`
		Time float64 `json:"time"`

		// For error responses
		Error string `json:"error,omitempty"`

		// For begin queries
		TxID string `json:"txId,omitempty"`

		// For write queries
		LastInsertID int64 `json:"lastInsertId,omitempty"`
		RowsAffected int64 `json:"rowsAffected,omitempty"`

		// For read and write queries that return rows
		Columns []string `json:"columns,omitempty"`
		Types   []string `json:"types,omitempty"`
		Rows    [][]any  `json:"rows,omitempty"`
	}
)
