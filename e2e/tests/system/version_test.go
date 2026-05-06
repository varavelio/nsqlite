package system_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestVersionEndpointReturnsBuiltVersion(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Get(t, "/version", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "0.0.0-dev", string(response.Body))
}
