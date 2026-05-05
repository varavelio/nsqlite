package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nsqlite/nsqlite/internal/nsqlite/db"
	"github.com/nsqlite/nsqlite/internal/nsqlite/log"
	"github.com/nsqlite/nsqlite/internal/nsqlite/sqlitec"
	"github.com/nsqlite/nsqlite/internal/util/httputil"
)

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
	TxID   string               `json:"txId"`
	Query  string               `json:"query"`
	Params []sqlitec.QueryParam `json:"params"`
}

// queryHandler is the HTTP handler for the /query endpoint that
// executes SQL queries.
func (s *Server) queryHandler(w http.ResponseWriter, r *http.Request) error {
	s.DBStats.IncHTTPRequests()
	s.DBStats.IncQueuedHTTPRequests()
	defer s.DBStats.DecQueuedHTTPRequests()
	ctx := r.Context()

	var queries []Query
	if err := json.NewDecoder(r.Body).Decode(&queries); err != nil {
		return httputil.NewJSONError(
			http.StatusBadRequest, err, "Failed to read request body",
		)
	}

	allStart := time.Now()
	results := []ResponseResult{}

	for _, q := range queries {
		thisStart := time.Now()

		if q.Query == "" {
			results = append(results, ResponseResult{
				Type:  "error",
				Time:  time.Since(thisStart).Seconds(),
				Error: "Empty query",
			})
			s.Logger.ErrorNs(log.NsServer, "Error executing query", log.KV{
				"query":  q.Query,
				"params": q.Params,
				"txId":   q.TxID,
				"error":  "Empty query",
			})
			continue
		}

		res, err := s.DB.Query(ctx, db.Query{
			TxID:   q.TxID,
			Query:  q.Query,
			Params: q.Params,
		})
		if err != nil {
			results = append(results, ResponseResult{
				Type:  "error",
				Time:  time.Since(thisStart).Seconds(),
				Error: err.Error(),
			})
			s.Logger.ErrorNs(log.NsServer, "Error executing query", log.KV{
				"query":  q.Query,
				"params": q.Params,
				"txId":   q.TxID,
				"error":  err.Error(),
			})
			continue
		}

		if res.Type == db.QueryTypeBegin {
			results = append(results, ResponseResult{
				Type: "begin",
				Time: time.Since(thisStart).Seconds(),
				TxID: res.TxID,
			})
			continue
		}

		if res.Type == db.QueryTypeCommit {
			results = append(results, ResponseResult{
				Type: "commit",
				Time: time.Since(thisStart).Seconds(),
			})
			continue
		}

		if res.Type == db.QueryTypeRollback {
			results = append(results, ResponseResult{
				Type: "rollback",
				Time: time.Since(thisStart).Seconds(),
			})
			continue
		}

		if res.Type == db.QueryTypeWrite {
			results = append(results, ResponseResult{
				Type:         "write",
				Time:         time.Since(thisStart).Seconds(),
				LastInsertID: res.LastInsertID,
				RowsAffected: res.RowsAffected,
				Columns:      res.Columns,
				Types:        res.Types,
				Rows:         res.Rows,
			})
			continue
		}

		if res.Type == db.QueryTypeRead {
			results = append(results, ResponseResult{
				Type:    "read",
				Time:    time.Since(thisStart).Seconds(),
				Columns: res.Columns,
				Types:   res.Types,
				Rows:    res.Rows,
			})
			continue
		}

		results = append(results, ResponseResult{
			Type:  "error",
			Time:  time.Since(thisStart).Seconds(),
			Error: "Unknown query response type: " + res.Type.Value,
		})
		s.Logger.ErrorNs(log.NsServer, "Unknown query response type", log.KV{
			"query":  q.Query,
			"params": q.Params,
			"txId":   q.TxID,
			"type":   res.Type.Value,
		})
	}

	return httputil.WriteJSON(w, http.StatusOK, Response{
		Time:    time.Since(allStart).Seconds(),
		Results: results,
	})
}
