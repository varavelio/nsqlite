package server

import (
	"context"
	"time"

	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/vdl"
)

// databaseQueryProc handles the Database.query RPC procedure.
// It executes each query in the request sequentially and aggregates the results.
func (s *Server) databaseQueryProc(
	c *vdl.DatabaseQueryHandlerContext[requestProps],
) (vdl.DatabaseQueryOutput, error) {
	return s.executeQueries(c.Context, c.Props, c.Input.Queries), nil
}

// executeQueries executes queries sequentially and aggregates their results.
func (s *Server) executeQueries(
	ctx context.Context,
	props requestProps,
	queries []vdl.Query,
) vdl.DatabaseQueryOutput {
	allStartedAt := time.Now()
	results := make([]vdl.QueryResult, 0, len(queries))

	for _, query := range queries {
		result := s.executeRequestQuery(ctx, props, query)
		results = append(results, result)
	}

	return vdl.DatabaseQueryOutput{
		Time:    time.Since(allStartedAt).Seconds(),
		Results: results,
	}
}

// executeRequestQuery executes a single query from the RPC request.
func (s *Server) executeRequestQuery(
	ctx context.Context,
	props requestProps,
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

		s.Logger.Error(ctx, "error executing query",
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

		s.Logger.Error(ctx, "error executing query",
			"query", query.Query,
			"params", query.Params,
			"txId", vdl.Val(query.TxId),
			"error", err.Error(),
		)

		return result
	}

	result, err := s.DB.Query(ctx, db.Query{
		TxID:    vdl.Val(query.TxId),
		Query:   query.Query,
		Params:  params,
		Role:    authorizerRoleForAuthRole(props.Role),
		TxOwner: props.Principal,
	})
	if err != nil {
		errorMessage := err.Error()
		queryResult := vdl.QueryResult{
			Type:  vdl.QueryResultTypeError,
			Time:  time.Since(startedAt).Seconds(),
			Error: &errorMessage,
		}

		s.Logger.Error(ctx, "error executing query",
			"query", query.Query,
			"params", query.Params,
			"txId", vdl.Val(query.TxId),
			"error", err.Error(),
		)

		return queryResult
	}

	return queryResultFromDB(startedAt, result)
}
