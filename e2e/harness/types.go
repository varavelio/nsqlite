package harness

// RPCError matches one VDL RPC error payload.
type RPCError struct {
	Message  string         `json:"message"`
	Category string         `json:"category,omitempty"`
	Code     string         `json:"code,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// RPCResponse matches the VDL RPC response envelope.
type RPCResponse[T any] struct {
	OK     bool     `json:"ok"`
	Output T        `json:"output"`
	Error  RPCError `json:"error"`
}

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

// LoadedStats matches the public `/stats` response payload.
type LoadedStats struct {
	StartedAt          string `json:"startedAt"`
	Uptime             string `json:"uptime"`
	QueuedBegins       int64  `json:"queuedBegins"`
	QueuedWrites       int64  `json:"queuedWrites"`
	QueuedHTTPRequests int64  `json:"queuedHttpRequests"`
	Totals             Totals `json:"totals"`
	Stats              []Stat `json:"stats"`
}

// WithoutRuntime clears runtime-dependent fields to keep assertions deterministic.
func (s LoadedStats) WithoutRuntime() LoadedStats {
	s.StartedAt = ""
	s.Uptime = ""
	return s
}

// Totals stores aggregate counters returned by `/stats`.
type Totals struct {
	Reads        int64 `json:"reads"`
	Writes       int64 `json:"writes"`
	Begins       int64 `json:"begins"`
	Commits      int64 `json:"commits"`
	Rollbacks    int64 `json:"rollbacks"`
	Errors       int64 `json:"errors"`
	HTTPRequests int64 `json:"httpRequests"`
}

// Stat stores one per-minute bucket from the `/stats` response.
type Stat struct {
	Minute       string `json:"minute"`
	Reads        int64  `json:"reads"`
	Writes       int64  `json:"writes"`
	Begins       int64  `json:"begins"`
	Commits      int64  `json:"commits"`
	Rollbacks    int64  `json:"rollbacks"`
	Errors       int64  `json:"errors"`
	HTTPRequests int64  `json:"httpRequests"`
}
