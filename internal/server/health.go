package server

import (
	"fmt"

	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/vdl"
)

func (s *Server) systemHealthProc(
	c *vdl.SystemHealthHandlerContext[requestProps],
) (vdl.SystemHealthOutput, error) {
	_, err := s.DB.Query(c.Context, db.Query{Query: "SELECT 1"})
	if err != nil {
		return vdl.SystemHealthOutput{}, fmt.Errorf("failed to query the database: %w", err)
	}

	return vdl.SystemHealthOutput{
		Healthy:  true,
		Database: true,
		Message:  "OK",
	}, nil
}
