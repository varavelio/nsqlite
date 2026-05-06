package httputil

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// WaitForServer waits for the server at the given URL to become ready. The
// server is considered ready if it returns a 200 OK response for the given
// URL.
func WaitForServer(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("server at %s did not become ready within %s", url, timeout.String())
		case <-time.After(50 * time.Millisecond):
		}
	}
}
