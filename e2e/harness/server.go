package harness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/internal/vdl"
)

const serverReadyTimeout = 10 * time.Second

const maxServerStartAttempts = 5

const (
	DatabaseQueryPath = "/rpc/Database/query"
	SystemHealthPath  = "/rpc/System/health"
	SystemSessionPath = "/rpc/System/session"
	SystemStatusPath  = "/rpc/System/status"
)

// ServerConfig defines the NSQLite process configuration for one E2E test.
type ServerConfig struct {
	AuthToken     string
	AuthTokenRW   string
	AuthTokenRO   string
	TxIdleTimeout time.Duration
}

// Server represents one running NSQLite process under test.
type Server struct {
	baseURL string
	dataDir string

	client  *http.Client
	cmd     *exec.Cmd
	exitCh  chan struct{}
	exitErr error

	stdout bytes.Buffer
	stderr bytes.Buffer

	stopOnce sync.Once
	stopErr  error
}

// StartServer starts one isolated NSQLite process for the current test.
func StartServer(t testing.TB, cfg ServerConfig) *Server {
	t.Helper()

	binaryPath := buildBinary(t)
	dataDir := t.TempDir()
	for attempt := range maxServerStartAttempts {
		server := startServerAttempt(t, binaryPath, dataDir, getFreePort(t), cfg)
		if err := server.waitUntilReady(); err != nil {
			_ = server.Stop()
			if attempt+1 < maxServerStartAttempts && isAddressAlreadyInUseError(err) {
				continue
			}
			require.NoError(t, err)
		}

		t.Cleanup(func() {
			require.NoError(t, server.Stop())
		})

		return server
	}

	t.Fatalf("failed to start server after %d attempts", maxServerStartAttempts)
	return nil
}

func startServerAttempt(
	t testing.TB,
	binaryPath, dataDir string,
	port int,
	cfg ServerConfig,
) *Server {
	t.Helper()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	cmd := exec.CommandContext(t.Context(), binaryPath)
	server := &Server{
		baseURL: baseURL,
		dataDir: dataDir,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		cmd:    cmd,
		exitCh: make(chan struct{}),
	}

	cmd.Dir = repoRoot()
	cmd.Env = append(os.Environ(),
		"NSQLITE_DATA_DIR="+dataDir,
		"NSQLITE_LISTEN_HOST=127.0.0.1",
		fmt.Sprintf("NSQLITE_LISTEN_PORT=%d", port),
	)
	if cfg.AuthToken != "" {
		cmd.Env = append(cmd.Env, "NSQLITE_AUTH_TOKEN="+cfg.AuthToken)
	}
	if cfg.AuthTokenRW != "" {
		cmd.Env = append(cmd.Env, "NSQLITE_AUTH_TOKEN_RW="+cfg.AuthTokenRW)
	}
	if cfg.AuthTokenRO != "" {
		cmd.Env = append(cmd.Env, "NSQLITE_AUTH_TOKEN_RO="+cfg.AuthTokenRO)
	}
	if cfg.TxIdleTimeout > 0 {
		cmd.Env = append(cmd.Env, "NSQLITE_TX_IDLE_TIMEOUT="+cfg.TxIdleTimeout.String())
	}
	cmd.Stdout = &server.stdout
	cmd.Stderr = &server.stderr

	require.NoError(t, cmd.Start())
	go func() {
		server.exitErr = normalizeExitError(cmd.Wait())
		close(server.exitCh)
	}()

	return server
}

func isAddressAlreadyInUseError(err error) bool {
	return strings.Contains(err.Error(), "address already in use")
}

// BaseURL returns the HTTP base URL for the running server.
func (s *Server) BaseURL() string {
	return s.baseURL
}

// DataDir returns the temporary data directory assigned to the server.
func (s *Server) DataDir() string {
	return s.dataDir
}

// Stop shuts down the server process.
func (s *Server) Stop() error {
	s.stopOnce.Do(func() {
		select {
		case <-s.exitCh:
			s.stopErr = s.exitErr
			return
		default:
		}

		if err := s.cmd.Process.Signal(
			syscall.SIGTERM,
		); err != nil &&
			!errors.Is(err, os.ErrProcessDone) {
			s.stopErr = fmt.Errorf("send SIGTERM: %w", err)
			return
		}

		select {
		case <-s.exitCh:
			s.stopErr = s.exitErr
		case <-time.After(5 * time.Second):
			if killErr := s.cmd.Process.Kill(); killErr != nil &&
				!errors.Is(killErr, os.ErrProcessDone) {
				s.stopErr = fmt.Errorf("kill process after timeout: %w", killErr)
				return
			}
			<-s.exitCh
			s.stopErr = s.exitErr
		}
	})

	return s.stopErr
}

// Get sends a GET request to the running server.
func (s *Server) Get(t testing.TB, path, token string) HTTPResponse {
	t.Helper()
	return s.do(t, http.MethodGet, path, nil, token)
}

// PostJSON sends a JSON POST request to the running server.
func (s *Server) PostJSON(t testing.TB, path string, body any, token string) HTTPResponse {
	t.Helper()

	encodedBody, err := json.Marshal(body)
	require.NoError(t, err)

	return s.do(t, http.MethodPost, path, bytes.NewReader(encodedBody), token)
}

// Query sends one or more queries to `/query` and decodes the successful response.
func (s *Server) Query(t testing.TB, token string, queries ...Query) QueryResponse {
	t.Helper()

	response := s.QueryResponse(t, token, queries...)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	return DecodeQueryResponse(t, response).WithoutTiming()
}

// Stats fetches `/stats` and decodes the successful response.
func (s *Server) Stats(t testing.TB, token string) LoadedStats {
	t.Helper()

	response := s.StatusResponse(t, token)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	rpcResponse := DecodeJSON[RPCResponse[vdl.SystemStatusOutput]](t, response)
	require.True(t, rpcResponse.OK, "unexpected RPC error: %s", string(response.Body))

	return loadedStatsFromVDL(rpcResponse.Output.Stats)
}

// Version fetches `/rpc/System/status` and returns the reported version.
func (s *Server) Version(t testing.TB, token string) string {
	t.Helper()

	response := s.StatusResponse(t, token)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	rpcResponse := DecodeJSON[RPCResponse[vdl.SystemStatusOutput]](t, response)
	require.True(t, rpcResponse.OK, "unexpected RPC error: %s", string(response.Body))

	return rpcResponse.Output.Version
}

// SessionRole fetches `/rpc/System/session` and returns the authenticated role.
func (s *Server) SessionRole(t testing.TB, token string) string {
	t.Helper()

	response := s.SessionResponse(t, token)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	rpcResponse := DecodeJSON[RPCResponse[vdl.SystemSessionOutput]](t, response)
	require.True(t, rpcResponse.OK, "unexpected RPC error: %s", string(response.Body))

	return string(rpcResponse.Output.Role)
}

// QueryResponse sends one or more queries to the Database.query RPC and returns the raw HTTP response.
func (s *Server) QueryResponse(t testing.TB, token string, queries ...Query) HTTPResponse {
	t.Helper()

	return s.PostJSON(t, DatabaseQueryPath, map[string]any{
		"queries": toRPCQueries(queries),
	}, token)
}

// HealthResponse calls the System.health RPC and returns the raw HTTP response.
func (s *Server) HealthResponse(t testing.TB, token string) HTTPResponse {
	t.Helper()
	return s.PostJSON(t, SystemHealthPath, map[string]any{}, token)
}

// SessionResponse calls the System.session RPC and returns the raw HTTP response.
func (s *Server) SessionResponse(t testing.TB, token string) HTTPResponse {
	t.Helper()
	return s.PostJSON(t, SystemSessionPath, map[string]any{}, token)
}

// StatusResponse calls the System.status RPC and returns the raw HTTP response.
func (s *Server) StatusResponse(t testing.TB, token string) HTTPResponse {
	t.Helper()
	return s.PostJSON(t, SystemStatusPath, map[string]any{}, token)
}

func (s *Server) waitUntilReady() error {
	deadline := time.Now().Add(serverReadyTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-s.exitCh:
			return fmt.Errorf(
				"server exited before becoming ready: %w\nstdout:\n%s\nstderr:\n%s",
				s.exitErr,
				s.stdout.String(),
				s.stderr.String(),
			)
		default:
		}

		body := bytes.NewReader([]byte(`{}`))
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			s.baseURL+SystemHealthPath,
			body,
		)
		if err != nil {
			return fmt.Errorf("build readiness request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf(
		"server did not become ready within %s\nstdout:\n%s\nstderr:\n%s",
		serverReadyTimeout,
		s.stdout.String(),
		s.stderr.String(),
	)
}

func (s *Server) do(t testing.TB, method, path string, body io.Reader, token string) HTTPResponse {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, s.baseURL+path, body)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       respBody,
	}
}

func toRPCQueries(queries []Query) []map[string]any {
	rpcQueries := make([]map[string]any, 0, len(queries))
	for _, query := range queries {
		rpcQuery := map[string]any{
			"query": query.Query,
		}
		if query.TxID != "" {
			rpcQuery["txId"] = query.TxID
		}
		if len(query.Params) > 0 {
			rpcParams := make([]map[string]any, 0, len(query.Params))
			for _, param := range query.Params {
				rpcParam := map[string]any{
					"value": toRPCValue(param.Value),
				}
				if param.Name != "" {
					rpcParam["name"] = param.Name
				}
				rpcParams = append(rpcParams, rpcParam)
			}
			rpcQuery["params"] = rpcParams
		}
		rpcQueries = append(rpcQueries, rpcQuery)
	}
	return rpcQueries
}

func toRPCValue(value any) map[string]any {
	switch v := value.(type) {
	case nil:
		return map[string]any{"null": true}
	case bool:
		if v {
			return map[string]any{"integer": 1}
		}
		return map[string]any{"integer": 0}
	case int:
		return map[string]any{"integer": v}
	case int8:
		return map[string]any{"integer": v}
	case int16:
		return map[string]any{"integer": v}
	case int32:
		return map[string]any{"integer": v}
	case int64:
		return map[string]any{"integer": v}
	case uint:
		return map[string]any{"integer": v}
	case uint8:
		return map[string]any{"integer": v}
	case uint16:
		return map[string]any{"integer": v}
	case uint32:
		return map[string]any{"integer": v}
	case uint64:
		return map[string]any{"integer": v}
	case float32:
		return map[string]any{"real": v}
	case float64:
		return map[string]any{"real": v}
	case string:
		return map[string]any{"text": v}
	default:
		return map[string]any{"unsupported": v}
	}
}

func queryResponseFromVDL(output vdl.DatabaseQueryOutput) QueryResponse {
	response := QueryResponse{
		Time:    output.Time,
		Results: make([]QueryResult, 0, len(output.Results)),
	}
	for _, result := range output.Results {
		response.Results = append(response.Results, queryResultFromVDL(result))
	}
	return response
}

// DecodeQueryResponse decodes a successful Database.query RPC response into the legacy test shape.
func DecodeQueryResponse(t testing.TB, response HTTPResponse) QueryResponse {
	t.Helper()

	decoded, err := DecodeQueryResponseBody(response.Body)
	require.NoError(t, err, "unexpected response body: %s", string(response.Body))

	return decoded
}

// DecodeQueryResponseBody decodes a successful Database.query RPC response body into the legacy test shape.
func DecodeQueryResponseBody(body []byte) (QueryResponse, error) {
	var rpcResponse RPCResponse[vdl.DatabaseQueryOutput]
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return QueryResponse{}, err
	}
	if !rpcResponse.OK {
		return QueryResponse{}, fmt.Errorf("rpc error: %s", rpcResponse.Error.Message)
	}

	return queryResponseFromVDL(rpcResponse.Output), nil
}

func queryResultFromVDL(result vdl.QueryResult) QueryResult {
	converted := QueryResult{
		Type: string(result.Type),
		Time: result.Time,
	}
	if result.Error != nil {
		converted.Error = *result.Error
	}
	if result.TxId != nil {
		converted.TxID = *result.TxId
	}
	if result.LastInsertId != nil {
		converted.LastInsertID = *result.LastInsertId
	}
	if result.RowsAffected != nil {
		converted.RowsAffected = *result.RowsAffected
	}
	if result.Columns != nil {
		converted.Columns = append([]string(nil), (*result.Columns)...)
	}
	if result.Types != nil {
		converted.Types = make([]string, 0, len(*result.Types))
		for _, typ := range *result.Types {
			converted.Types = append(converted.Types, string(typ))
		}
	}
	if result.Rows != nil {
		converted.Rows = make([][]any, 0, len(*result.Rows))
		for _, row := range *result.Rows {
			convertedRow := make([]any, 0, len(row))
			for _, value := range row {
				convertedRow = append(convertedRow, sqliteValueToAny(value))
			}
			converted.Rows = append(converted.Rows, convertedRow)
		}
	}
	return converted
}

func sqliteValueToAny(value vdl.SqliteValue) any {
	if value.Null != nil && *value.Null {
		return nil
	}
	if value.Integer != nil {
		return float64(*value.Integer)
	}
	if value.Real != nil {
		return *value.Real
	}
	if value.Text != nil {
		return *value.Text
	}
	if value.Blob != nil {
		return *value.Blob
	}
	return nil
}

func loadedStatsFromVDL(stats vdl.Stats) LoadedStats {
	loaded := LoadedStats{
		StartedAt: stats.StartedAt.Format(time.RFC3339),
		Uptime: (time.Duration(stats.UptimeSeconds * float64(time.Second))).Round(time.Second).
			String(),
		QueuedBegins:       stats.Queued.Begins,
		QueuedWrites:       stats.Queued.Writes,
		QueuedHTTPRequests: stats.Queued.HttpRequests,
		Totals: Totals{
			Reads:        stats.Totals.Reads,
			Writes:       stats.Totals.Writes,
			Begins:       stats.Totals.Begins,
			Commits:      stats.Totals.Commits,
			Rollbacks:    stats.Totals.Rollbacks,
			Errors:       stats.Totals.Errors,
			HTTPRequests: stats.Totals.HttpRequests,
		},
		Stats: make([]Stat, 0, len(stats.Minutes)),
	}
	for minute, totals := range stats.Minutes {
		loaded.Stats = append(loaded.Stats, Stat{
			Minute:       minute,
			Reads:        totals.Reads,
			Writes:       totals.Writes,
			Begins:       totals.Begins,
			Commits:      totals.Commits,
			Rollbacks:    totals.Rollbacks,
			Errors:       totals.Errors,
			HTTPRequests: totals.HttpRequests,
		})
	}
	sort.Slice(loaded.Stats, func(i, j int) bool {
		return loaded.Stats[j].Minute < loaded.Stats[i].Minute
	})
	return loaded
}

// HTTPResponse stores the raw HTTP response returned by the server.
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// DecodeJSON decodes a JSON response body into the requested type.
func DecodeJSON[T any](t testing.TB, response HTTPResponse) T {
	t.Helper()

	var value T
	require.NoError(
		t,
		json.Unmarshal(response.Body, &value),
		"response body: %s",
		string(response.Body),
	)
	return value
}

func getFreePort(t testing.TB) int {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return address.Port
}

func repoRoot() string {
	_, fileName, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to resolve harness location")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(fileName), "..", ".."))
}

func normalizeExitError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
		return nil
	}

	return err
}
