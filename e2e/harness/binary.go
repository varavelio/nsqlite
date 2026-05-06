package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	binaryPathOnce sync.Once
	binaryPath     string
	binaryPathErr  error
)

// BinaryPath returns the cached NSQLite test binary path.
func BinaryPath(t testing.TB) string {
	t.Helper()

	binaryPathOnce.Do(func() {
		cacheDir, err := os.MkdirTemp("", "nsqlite-e2e-*")
		if err != nil {
			binaryPathErr = fmt.Errorf("create temp binary dir: %w", err)
			return
		}

		binaryPath = filepath.Join(
			cacheDir,
			fmt.Sprintf("nsqlite-%s-%s", runtime.GOOS, runtime.GOARCH),
		)
		binaryPathErr = buildBinaryLocked(binaryPath)
	})

	require.NoError(t, binaryPathErr)
	return binaryPath
}

// buildBinary returns the cached NSQLite test binary path.
func buildBinary(t testing.TB) string {
	t.Helper()
	return BinaryPath(t)
}

func buildBinaryLocked(binaryPath string) error {
	tempBinaryPath := fmt.Sprintf("%s.tmp-%d", binaryPath, os.Getpid())
	defer func() { _ = os.Remove(tempBinaryPath) }()

	cmd := exec.CommandContext(
		context.Background(),
		"go",
		"build",
		"-o",
		tempBinaryPath,
		"./cmd/nsqlite",
	)
	cmd.Dir = repoRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build: %w\n%s", err, string(output))
	}
	if err := os.Rename(tempBinaryPath, binaryPath); err != nil {
		return fmt.Errorf("rename built binary: %w", err)
	}

	return nil
}
