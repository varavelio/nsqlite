package server

import (
	"errors"
	"fmt"
	"net/http"

	json "github.com/goccy/go-json"
	"github.com/varavelio/nsqlite/internal/util/httputil"
	"github.com/varavelio/nsqlite/internal/vdl"
)

// requestProps holds per-request metadata attached to each RPC handler context.
type requestProps struct {
	Role      authRole
	Principal string
}

// newRPCServer creates and configures the VDL RPC server with all registered procedures.
func (s *Server) newRPCServer() *vdl.Server[requestProps] {
	rpcServer := vdl.NewServer[requestProps]()
	rpcServer.SetErrorHandler(s.rpcErrorHandler)
	rpcServer.RPCs.Database().Procs.Query().Handle(s.databaseQueryProc)
	rpcServer.RPCs.System().Procs.Health().Handle(s.systemHealthProc)
	rpcServer.RPCs.System().Procs.Session().Handle(s.systemSessionProc)
	rpcServer.RPCs.System().Procs.Status().Handle(s.systemStatusProc)
	return rpcServer
}

// rpcHandler is the top-level HTTP handler for all RPC requests.
// It performs authentication, authorization, request tracking, and dispatches
// to the appropriate VDL procedure handler.
func (s *Server) rpcHandler(w http.ResponseWriter, r *http.Request) error {
	rpcName := r.PathValue("rpcName")
	operationName := r.PathValue("operationName")

	props, err := s.authorizeRPCRequest(r, rpcName, operationName)
	if err != nil {
		return s.writeRPCError(w, err)
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

// authorizeRPCRequest authenticates the request and enforces role-based access
// for protected RPC procedures.
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

// rpcErrorHandler is the VDL error handler callback.
// It logs the error and returns a safe message suitable for the API response.
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

// writeRPCError writes a VDL error response to the HTTP writer.
func (s *Server) writeRPCError(w http.ResponseWriter, err error) error {
	message := err.Error()
	var jsonErr httputil.JSONError
	if errors.As(err, &jsonErr) && jsonErr.SafeMessage != "" {
		message = jsonErr.SafeMessage
	}

	body, marshalErr := json.Marshal(vdl.Response[any]{
		Ok:    false,
		Error: vdl.Error{Message: message},
	})
	if marshalErr != nil {
		return fmt.Errorf("marshal rpc error response: %w", marshalErr)
	}

	return httputil.WriteJSONBytes(w, http.StatusOK, body)
}
