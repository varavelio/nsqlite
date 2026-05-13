package server

import (
	"time"

	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/vdl"
)

// databaseQueryProc handles the Database.query RPC procedure.
// It executes each query in the request sequentially and aggregates the results.
func (s *Server) databaseQueryProc(
	c *vdl.DatabaseQueryHandlerContext[requestProps],
) (vdl.DatabaseQueryOutput, error) {
	allStartedAt := time.Now()
	results := make([]vdl.QueryResult, 0, len(c.Input.Queries))

	for _, query := range c.Input.Queries {
		result := s.executeRequestQuery(c, query)
		results = append(results, result)
	}

	return vdl.DatabaseQueryOutput{
		Time:    time.Since(allStartedAt).Seconds(),
		Results: results,
	}, nil
}

// executeRequestQuery executes a single query from the RPC request.
func (s *Server) executeRequestQuery(
	c *vdl.DatabaseQueryHandlerContext[requestProps],
	query vdl.Query,
) vdl.QueryResult {
	startedAt := time.Now()

	if query.Query == "" {
		errorMessage := "Empty query"
		result := vdl.QueryResult{
			Type:  vdl.QueryResultTypeError,
			Time:  time.Since(startedAt).Seconds(),
			Error: &errorMessage,
		}

		s.Logger.Error(c.Context, "error executing query",
			"query", query.Query,
			"params", query.Params,
			"txId", vdl.Val(query.TxId),
			"error", "empty query",
		)

		return result
	}

	params, err := sqliteParamsFromVDL(query.Params)
	if err != nil {
		errorMessage := err.Error()
		result := vdl.QueryResult{
			Type:  vdl.QueryResultTypeError,
			Time:  time.Since(startedAt).Seconds(),
			Error: &errorMessage,
		}

		s.Logger.Error(c.Context, "error executing query",
			"query", query.Query,
			"params", query.Params,
			"txId", vdl.Val(query.TxId),
			"error", err.Error(),
		)

		return result
	}

	result, err := s.DB.Query(c.Context, db.Query{
		TxID:    vdl.Val(query.TxId),
		Query:   query.Query,
		Params:  params,
		Role:    authorizerRoleForAuthRole(c.Props.Role),
		TxOwner: c.Props.Principal,
	})
	if err != nil {
		errorMessage := err.Error()
		queryResult := vdl.QueryResult{
			Type:  vdl.QueryResultTypeError,
			Time:  time.Since(startedAt).Seconds(),
			Error: &errorMessage,
		}

		s.Logger.Error(c.Context, "error executing query",
			"query", query.Query,
			"params", query.Params,
			"txId", vdl.Val(query.TxId),
			"error", err.Error(),
		)

		return queryResult
	}

	return queryResultFromDB(startedAt, result)
}
