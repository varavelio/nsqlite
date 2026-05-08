package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/sqlite"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

func (s *Server) maxRequestSize() int64 {
	if s.MaxRequestSizeMB <= 0 {
		return 100 * 1024 * 1024 // 100MB default
	}
	return int64(s.MaxRequestSizeMB) * 1024 * 1024
}

// ResponseResult represents the structure of a query result.
type ResponseResult struct {
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

// Response represents the structure of an outgoing response.
type Response struct {
	Time    float64          `json:"time"`
	Results []ResponseResult `json:"results"`
}

// Query represents a single query within a request.
type Query struct {
	TxID   string              `json:"txId"`
	Query  string              `json:"query"`
	Params []sqlite.QueryParam `json:"params"`
}

// queryHandler decodes request queries, classifies each one once, enforces
// authorization, and executes the allowed queries.
func (s *Server) queryHandler(w http.ResponseWriter, r *http.Request) error {
	s.DBStats.IncHTTPRequests()
	s.DBStats.IncQueuedHTTPRequests()
	defer s.DBStats.DecQueuedHTTPRequests()

	ctx := r.Context()
	role, ok := getAuthRoleFromContext(ctx)
	if !ok {
		return httputil.NewJSONError(
			http.StatusInternalServerError,
			errors.New("missing auth role in request context"),
			"Internal server error",
		)
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize())

	queries, err := decodeQueries(r)
	if err != nil {
		return err
	}

	allStart := time.Now()
	results := make([]ResponseResult, 0, len(queries))

	for _, query := range queries {
		result, shouldStop := s.executeRequestQuery(ctx, role, query)
		if shouldStop {
			return forbiddenError()
		}

		results = append(results, result)
	}

	return httputil.WriteJSON(w, http.StatusOK, Response{
		Time:    time.Since(allStart).Seconds(),
		Results: results,
	})
}

// decodeQueries reads and validates the request body for the /query endpoint.
func decodeQueries(r *http.Request) ([]Query, error) {
	queries := []Query{}
	if err := json.NewDecoder(r.Body).Decode(&queries); err != nil {
		return nil, httputil.NewJSONError(
			http.StatusBadRequest,
			err,
			"Failed to read request body",
		)
	}

	return queries, nil
}

// executeRequestQuery sends a single request query to the database and formats
// the HTTP response payload.
func (s *Server) executeRequestQuery(
	ctx context.Context,
	role authRole,
	query Query,
) (ResponseResult, bool) {
	startedAt := time.Now()

	if query.Query == "" {
		return s.emptyQueryResult(ctx, query, startedAt), false
	}

	principal, _ := getAuthPrincipalFromContext(ctx)

	result, err := s.DB.Query(ctx, db.Query{
		TxID:    query.TxID,
		Query:   query.Query,
		Params:  query.Params,
		Role:    authorizerRoleForAuthRole(role),
		TxOwner: principal,
	})
	if err != nil {
		return s.executionErrorResult(ctx, query, startedAt, err), false
	}

	return buildResponseResult(startedAt, result), false
}

func authorizerRoleForAuthRole(role authRole) sqlite.AuthorizerRole {
	switch role {
	case authRoleReadWrite:
		return sqlite.AuthorizerRoleReadWrite
	case authRoleReadOnly:
		return sqlite.AuthorizerRoleReadOnly
	default:
		return sqlite.AuthorizerRoleAdmin
	}
}

// emptyQueryResult builds the response payload for an empty query.
func (s *Server) emptyQueryResult(
	ctx context.Context,
	query Query,
	startedAt time.Time,
) ResponseResult {
	result := ResponseResult{
		Type:  "error",
		Time:  time.Since(startedAt).Seconds(),
		Error: "Empty query",
	}

	s.Logger.Error(ctx, "error executing query",
		"query", query.Query,
		"params", query.Params,
		"txId", query.TxID,
		"error", "empty query",
	)

	return result
}

// executionErrorResult builds the response payload for a query execution failure.
func (s *Server) executionErrorResult(
	ctx context.Context,
	query Query,
	startedAt time.Time,
	err error,
) ResponseResult {
	result := ResponseResult{
		Type:  "error",
		Time:  time.Since(startedAt).Seconds(),
		Error: err.Error(),
	}

	s.Logger.Error(ctx, "error executing query",
		"query", query.Query,
		"params", query.Params,
		"txId", query.TxID,
		"error", err.Error(),
	)

	return result
}

// buildResponseResult converts a database query result into an HTTP response result.
func buildResponseResult(startedAt time.Time, result db.QueryResult) ResponseResult {
	base := ResponseResult{
		Type: string(result.Type),
		Time: time.Since(startedAt).Seconds(),
	}

	if result.Type == db.QueryTypeBegin {
		base.TxID = result.TxID
		return base
	}

	if result.Type == db.QueryTypeCommit || result.Type == db.QueryTypeRollback {
		return base
	}

	if result.Type == db.QueryTypeWrite {
		base.LastInsertID = result.LastInsertID
		base.RowsAffected = result.RowsAffected
		base.Columns = result.Columns
		base.Types = result.Types
		base.Rows = result.Rows
		return base
	}

	if result.Type == db.QueryTypeRead {
		base.Columns = result.Columns
		base.Types = result.Types
		base.Rows = result.Rows
		return base
	}

	base.Type = "error"
	base.Error = "Unknown query response type: " + string(result.Type)
	return base
}
