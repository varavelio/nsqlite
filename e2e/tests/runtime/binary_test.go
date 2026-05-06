package runtime_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestBinaryStartsWithDefaultsAndValidListenAndDataDir(t *testing.T) {
	t.Parallel()

	running := startBinaryWithRetry(t, harness.BinaryPath(t), t.TempDir())

	require.NoError(t, running.cmd.Process.Signal(syscall.SIGTERM))
	require.NoError(t, normalizeExitError(<-running.exitCh))
	require.Contains(t, running.stdout.String(), "starting NSQLite server")
	require.Empty(t, running.stderr.String())
	require.Contains(t, running.baseURL, "http://127.0.0.1:")
}

func TestBinaryRejectsInvalidPortViaCLI(t *testing.T) {
	t.Parallel()

	binaryPath := harness.BinaryPath(t)
	output, err := exec.CommandContext(
		t.Context(),
		binaryPath,
		"--data-dir", t.TempDir(),
		"--listen-port", "99999",
	).CombinedOutput()

	require.Error(t, err)
	require.Contains(t, string(output), "invalid listen port 99999, valid values are 1-65535")
}

func TestBinaryRejectsInvalidTransactionIdleTimeoutViaEnv(t *testing.T) {
	t.Parallel()

	binaryPath := harness.BinaryPath(t)
	cmd := exec.CommandContext(t.Context(), binaryPath)
	cmd.Env = append(os.Environ(),
		"NSQLITE_DATA_DIR="+t.TempDir(),
		"NSQLITE_LISTEN_PORT=12345",
		"NSQLITE_TX_IDLE_TIMEOUT=0s",
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "invalid transaction timeout 0s, must be greater than zero")
}

func freePort(t testing.TB) int {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return address.Port
}

func waitForHealth(
	baseURL string,
	exitCh <-chan error,
	stdout, stderr *bytes.Buffer,
	timeout time.Duration,
) (bool, error) {
	client := &http.Client{Timeout: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for ctx.Err() == nil {
		select {
		case err := <-exitCh:
			return true, fmt.Errorf(
				"binary exited before becoming ready: %w\nstdout:\n%s\nstderr:\n%s",
				err,
				stdout.String(),
				stderr.String(),
			)
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
		if err != nil {
			return false, err
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return false, nil
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	return false, fmt.Errorf("server at %s did not become ready within %s", baseURL, timeout)
}

type runningBinary struct {
	cmd     *exec.Cmd
	baseURL string
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
	exitCh  chan error
}

func startBinaryWithRetry(
	t testing.TB,
	binaryPath, dataDir string,
) *runningBinary {
	t.Helper()

	for range 5 {
		port := freePort(t)
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

		cmd := exec.CommandContext(
			t.Context(),
			binaryPath,
			"--data-dir", dataDir,
			"--listen-port", fmt.Sprint(port),
		)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		require.NoError(t, cmd.Start())
		exitCh := make(chan error, 1)
		go func() {
			exitCh <- cmd.Wait()
		}()

		exitConsumed, err := waitForHealth(baseURL, exitCh, stdout, stderr, 10*time.Second)
		if err == nil {
			return &runningBinary{
				cmd:     cmd,
				baseURL: baseURL,
				stdout:  stdout,
				stderr:  stderr,
				exitCh:  exitCh,
			}
		} else if !strings.Contains(stdout.String()+stderr.String()+err.Error(), "address already in use") {
			if !exitConsumed {
				_ = cmd.Process.Kill()
				_ = normalizeExitError(<-exitCh)
			}
			require.NoError(t, err)
		}

		if !exitConsumed {
			_ = cmd.Process.Kill()
			_ = normalizeExitError(<-exitCh)
		}
	}

	t.Fatalf("failed to start binary after retries")
	return nil
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
