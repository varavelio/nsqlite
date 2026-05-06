package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const serverReadyTimeout = 10 * time.Second

const maxServerStartAttempts = 5

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

	response := s.PostJSON(t, "/query", queries, token)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	return DecodeJSON[QueryResponse](t, response).WithoutTiming()
}

// Stats fetches `/stats` and decodes the successful response.
func (s *Server) Stats(t testing.TB, token string) LoadedStats {
	t.Helper()

	response := s.Get(t, "/stats", token)
	require.Equal(
		t,
		http.StatusOK,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	return DecodeJSON[LoadedStats](t, response)
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

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			s.baseURL+"/health",
			nil,
		)
		if err != nil {
			return fmt.Errorf("build readiness request: %w", err)
		}

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

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
		return nil
	}

	return err
}
