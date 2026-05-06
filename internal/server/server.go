package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/logger"
	"github.com/varavelio/nsqlite/internal/stats"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

// Config represents the configuration for a NSQLite server.
type Config struct {
	// Logger is the shared NSQLite logger.
	Logger logger.Logger
	// DBStats is the NSQLite database stats instance to use.
	DBStats *stats.DBStats
	// DB is the NSQLite database instance to use.
	DB *db.DB
	// AuthTokens are the admin auth tokens.
	AuthTokens []string
	// ReadWriteAuthTokens are the read/write auth tokens.
	ReadWriteAuthTokens []string
	// ReadOnlyAuthTokens are the read-only auth tokens.
	ReadOnlyAuthTokens []string
	// ListenHost is the host to listen on.
	ListenHost string
	// ListenPort is the port to listen on.
	ListenPort string
}

// Server is the server for NSQLite.
type Server struct {
	Config
	authTokens     []authToken
	authTokenCache sync.Map
	authTokenSalt  string
	isInitialized  bool
	server         http.Server
}

// NewServer creates a new NSQLite server.
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
		server:         http.Server{},
	}
	return &s, nil
}

// IsInitialized returns true if the server is initialized.
func (s *Server) IsInitialized() bool {
	return s.isInitialized
}

// createMux creates the HTTP mux for the server.
func (s *Server) createMux() *http.ServeMux {
	buildHandler := httputil.CreateHandlerFuncBuilder(s.errorHandler)
	mux := http.NewServeMux()

	headerAuthMws := []httputil.Middleware{
		s.adminAuthMiddleware,
	}

	routes := []struct {
		pattern     string
		handler     httputil.HandlerFuncErr
		middlewares []httputil.Middleware
	}{
		{
			pattern: "/health",
			handler: s.healthHandler,
		},
		{
			pattern:     "/version",
			handler:     s.versionHandler,
			middlewares: headerAuthMws,
		},
		{
			pattern:     "/stats",
			handler:     s.statsHandler,
			middlewares: headerAuthMws,
		},
		{
			pattern:     "/query",
			handler:     s.queryHandler,
			middlewares: []httputil.Middleware{s.queryHandlerAuthMiddleware},
		},
	}

	setResponseHeaders := func(next httputil.HandlerFuncErr) httputil.HandlerFuncErr {
		return func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("x-server", "NSQLite")
			return next(w, r)
		}
	}

	for _, route := range routes {
		route.middlewares = append(route.middlewares, setResponseHeaders)
		mux.HandleFunc(
			route.pattern, buildHandler(route.handler, route.middlewares...),
		)
	}

	return mux
}

// authIsDisabled reports whether the server should allow every request.
func (s *Server) authIsDisabled() bool {
	return len(s.authTokens) == 0
}

// Start starts the server.
func (s *Server) Start() error {
	mux := s.createMux()
	addr := fmt.Sprintf("%s:%s", s.ListenHost, s.ListenPort)
	localAddr := fmt.Sprintf("http://%s:%s", "localhost", s.ListenPort)
	s.server = http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.Logger.Info(context.Background(), "server started at "+localAddr,
		"listenHost", s.ListenHost,
		"listenPort", s.ListenPort,
	)

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop() error {
	return s.server.Shutdown(context.TODO())
}
