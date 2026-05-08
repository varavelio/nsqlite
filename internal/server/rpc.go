package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/varavelio/nsqlite/internal/util/httputil"
	"github.com/varavelio/nsqlite/internal/vdl"
)

type requestProps struct {
	Role      authRole
	Principal string
}

func (s *Server) newRPCServer() *vdl.Server[requestProps] {
	rpcServer := vdl.NewServer[requestProps]()
	rpcServer.SetErrorHandler(s.rpcErrorHandler)
	rpcServer.RPCs.Database().Procs.Query().Handle(s.databaseQueryProc)
	rpcServer.RPCs.System().Procs.Health().Handle(s.systemHealthProc)
	rpcServer.RPCs.System().Procs.Session().Handle(s.systemSessionProc)
	rpcServer.RPCs.System().Procs.Status().Handle(s.systemStatusProc)
	return rpcServer
}

func (s *Server) rpcHandler(w http.ResponseWriter, r *http.Request) error {
	rpcName := r.PathValue("rpcName")
	operationName := r.PathValue("operationName")

	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize())

	props, err := s.authorizeRPCRequest(r, rpcName, operationName)
	if err != nil {
		return err
	}

	if rpcName == "Database" && operationName == "query" {
		s.DBStats.IncHTTPRequests()
		s.DBStats.IncQueuedHTTPRequests()
		defer s.DBStats.DecQueuedHTTPRequests()
	}

	if err := s.rpcServer.HandleRequest(
		r.Context(),
		props,
		rpcName,
		operationName,
		vdl.NewNetHTTPAdapter(w, r),
	); err != nil {
		return fmt.Errorf("handle rpc request: %w", err)
	}

	return nil
}

func (s *Server) authorizeRPCRequest(
	r *http.Request,
	rpcName string,
	operationName string,
) (requestProps, error) {
	if rpcName == "System" && operationName == "health" {
		return requestProps{Role: authRoleAdmin}, nil
	}

	role, principal, err := s.authenticateRequest(r)
	if err != nil {
		return requestProps{}, err
	}

	if rpcName == "System" && operationName == "status" && role != authRoleAdmin {
		return requestProps{}, forbiddenError()
	}

	return requestProps{Role: role, Principal: principal}, nil
}

func (s *Server) rpcErrorHandler(c *vdl.HandlerContext[requestProps, any], err error) vdl.Error {
	message := err.Error()
	var jsonErr httputil.JSONError
	if errors.As(err, &jsonErr) && jsonErr.SafeMessage != "" {
		message = jsonErr.SafeMessage
	}

	s.Logger.Error(c.Context, "rpc handler error",
		"rpc", c.RPCName(),
		"operation", c.OperationName(),
		"error", err.Error(),
	)

	return vdl.Error{Message: message}
}
