package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// buildBinary returns the cached NSQLite test binary path.
func buildBinary(t testing.TB) string {
	t.Helper()

	cacheDir := filepath.Join(os.TempDir(), "nsqlite-e2e")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))

	binaryPath := filepath.Join(
		cacheDir,
		fmt.Sprintf("nsqlite-%s-%s", runtime.GOOS, runtime.GOARCH),
	)
	lockPath := binaryPath + ".lock"

	for {
		if fileExists(binaryPath) {
			return binaryPath
		}

		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = lockFile.Close()
			defer func() { _ = os.Remove(lockPath) }()
			buildBinaryLocked(t, binaryPath)
			return binaryPath
		}

		if !errors.Is(err, os.ErrExist) {
			require.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func buildBinaryLocked(t testing.TB, binaryPath string) {
	t.Helper()

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
	require.NoError(t, err, "go build output:\n%s", string(output))
	require.NoError(t, os.Rename(tempBinaryPath, binaryPath))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
