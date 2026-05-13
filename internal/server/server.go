package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/logger"
	"github.com/varavelio/nsqlite/internal/stats"
	"github.com/varavelio/nsqlite/internal/util/httputil"
	"github.com/varavelio/nsqlite/internal/vdl"
)

// Config holds the configuration needed to create and start an NSQLite server.
type Config struct {
	Logger              logger.Logger
	DBStats             *stats.DBStats
	DB                  *db.DB
	AuthTokens          []string
	ReadWriteAuthTokens []string
	ReadOnlyAuthTokens  []string
	ListenHost          string
	ListenPort          string
	MaxRequestSizeMB    int
	IdleTimeout         time.Duration
}

// Server is the HTTP server for NSQLite.
// It handles RPC requests, manages authentication, and exposes the SQLite database
// over HTTP through a VDL-based RPC layer.
type Server struct {
	Config
	authTokens     []authToken
	authTokenCache sync.Map
	authTokenSalt  string
	isInitialized  bool
	rpcServer      *vdl.Server[requestProps]
	httpServer     http.Server
}

// NewServer creates a new NSQLite server from the given configuration.
// Defaults for ListenHost ("0.0.0.0") and ListenPort ("9876") are applied
// when the corresponding fields are empty.
func NewServer(config Config) (*Server, error) {
	if config.ListenHost == "" {
		config.ListenHost = "0.0.0.0"
	}
	if config.ListenPort == "" {
		config.ListenPort = "9876"
	}

	s := Server{
		Config: config,
		authTokens: newAuthTokens(
			config.AuthTokens,
			config.ReadWriteAuthTokens,
			config.ReadOnlyAuthTokens,
		),
		authTokenCache: sync.Map{},
		authTokenSalt:  uuid.NewString(),
		isInitialized:  true,
		rpcServer:      nil,
		httpServer:     http.Server{},
	}
	s.rpcServer = s.newRPCServer()
	return &s, nil
}

// IsInitialized reports whether the server has been properly initialized.
func (s *Server) IsInitialized() bool {
	return s.isInitialized
}

// Start begins serving HTTP requests on the configured address.
// It blocks until the server is stopped via Stop or a fatal error occurs.
func (s *Server) Start() error {
	mux := s.createMux()
	addr := fmt.Sprintf("%s:%s", s.ListenHost, s.ListenPort)
	localAddr := fmt.Sprintf("http://%s:%s", "localhost", s.ListenPort)
	s.httpServer = http.Server{
		Addr:        addr,
		Handler:     s.maxRequestBodyMiddleware(mux),
		IdleTimeout: s.IdleTimeout,
	}

	s.Logger.Info(context.Background(), "server started at "+localAddr,
		"listenHost", s.ListenHost,
		"listenPort", s.ListenPort,
	)

	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop() error {
	return s.httpServer.Shutdown(context.TODO())
}

// createMux builds the HTTP ServeMux and registers the RPC route.
func (s *Server) createMux() *http.ServeMux {
	buildHandler := httputil.CreateHandlerFuncBuilder(s.errorHandler)
	mux := http.NewServeMux()

	// NSQLite RPC endpoint
	mux.HandleFunc("POST /rpc/{rpcName}/{operationName}", buildHandler(s.rpcHandler))

	// NSQLite <-> RQLite compatibility endpoints
	mux.HandleFunc("GET /db/query", buildHandler(s.rqliteQueryHandler))
	mux.HandleFunc("POST /db/query", buildHandler(s.rqliteQueryHandler))
	mux.HandleFunc("POST /db/execute", buildHandler(s.rqliteExecuteHandler))
	mux.HandleFunc("POST /db/request", buildHandler(s.rqliteRequestHandler))

	return mux
}

// authIsDisabled reports whether authentication is skipped entirely.
func (s *Server) authIsDisabled() bool {
	return len(s.authTokens) == 0
}
