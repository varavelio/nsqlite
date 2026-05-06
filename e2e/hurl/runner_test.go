package hurl_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

var hurlServerConfig = harness.ServerConfig{
	AuthToken:   "admin-token",
	AuthTokenRW: "rw-token",
	AuthTokenRO: "ro-token",
}

func TestHurlScenarios(t *testing.T) {
	hurlRoot := hurlRootDir(t)
	hurlFiles := discoverHurlFiles(t, hurlRoot)
	if len(hurlFiles) == 0 {
		t.Skip("no Hurl scenarios found")
	}

	for _, hurlFile := range hurlFiles {
		relativePath, err := filepath.Rel(hurlRoot, hurlFile)
		require.NoError(t, err)

		t.Run(filepath.ToSlash(relativePath), func(t *testing.T) {
			t.Parallel()

			server := harness.StartServer(t, hurlServerConfig)
			runHurlFile(t, hurlRoot, hurlFile, server.BaseURL())
		})
	}
}

func hurlRootDir(t testing.TB) string {
	t.Helper()

	_, fileName, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to resolve Hurl test location")

	return filepath.Dir(fileName)
}

func discoverHurlFiles(t testing.TB, root string) []string {
	t.Helper()

	var hurlFiles []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".hurl" {
			return nil
		}

		hurlFiles = append(hurlFiles, path)
		return nil
	})
	require.NoError(t, err)

	sort.Strings(hurlFiles)
	return hurlFiles
}

func runHurlFile(t testing.TB, hurlRoot, hurlFile, baseURL string) {
	t.Helper()

	commandArgs := []string{
		"--test",
		"--no-color",
		"--no-output",
		"--error-format",
		"long",
		"--variable",
		"host=" + baseURL,
		"--variable",
		"admin_token=" + hurlServerConfig.AuthToken,
		"--variable",
		"rw_token=" + hurlServerConfig.AuthTokenRW,
		"--variable",
		"ro_token=" + hurlServerConfig.AuthTokenRO,
		hurlFile,
	}

	cmd := exec.CommandContext(t.Context(), "hurl", commandArgs...)
	cmd.Dir = hurlRoot
	output, err := cmd.CombinedOutput()
	require.NoError(
		t,
		err,
		"hurl failed for %s:\n%s",
		filepath.ToSlash(strings.TrimPrefix(hurlFile, hurlRoot+string(filepath.Separator))),
		formatHurlOutput(output),
	)
}

func formatHurlOutput(output []byte) string {
	trimmedOutput := strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return "(no output)"
	}
	return fmt.Sprintf("%s\n", trimmedOutput)
}
