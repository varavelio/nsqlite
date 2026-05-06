package harness

// Query matches the public `/query` request payload.
type Query struct {
	TxID   string       `json:"txId,omitempty"`
	Query  string       `json:"query"`
	Params []QueryParam `json:"params,omitempty"`
}

// QueryParam matches one query parameter in the public HTTP API.
type QueryParam struct {
	Name  string `json:"name,omitempty"`
	Value any    `json:"value"`
}

// QueryResponse matches the public `/query` response payload.
type QueryResponse struct {
	Time    float64       `json:"time"`
	Results []QueryResult `json:"results"`
}

// QueryResult matches one query result in the public `/query` response.
type QueryResult struct {
	Type         string   `json:"type"`
	Time         float64  `json:"time"`
	Error        string   `json:"error,omitempty"`
	TxID         string   `json:"txId,omitempty"`
	LastInsertID int64    `json:"lastInsertId,omitempty"`
	RowsAffected int64    `json:"rowsAffected,omitempty"`
	Columns      []string `json:"columns,omitempty"`
	Types        []string `json:"types,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
}

// WithoutTiming clears timing fields to keep assertions deterministic.
func (r QueryResponse) WithoutTiming() QueryResponse {
	r.Time = 0
	for i := range r.Results {
		r.Results[i].Time = 0
	}
	return r
}

// APIError matches the public JSON error payload returned by NSQLite.
type APIError struct {
	ID      string `json:"id"`
	Error   string `json:"error"`
	Message string `json:"message"`
}
