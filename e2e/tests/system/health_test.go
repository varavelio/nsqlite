package system_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestHealthEndpointReturnsOK(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	response := server.Get(t, "/health", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "OK", string(response.Body))
}
