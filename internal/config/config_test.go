package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("fails when no auth tokens are configured", func(t *testing.T) {
		_, err := Parse([]string{"--data-dir", t.TempDir()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one authentication token")
	})

	t.Run("parses multiple tokens per role", func(t *testing.T) {
		cfg, err := Parse([]string{
			"--data-dir", t.TempDir(),
			"--auth-token", "admin-1,admin-2",
			"--auth-token-rw", "rw-1,rw-2",
			"--auth-token-ro", "ro-1,ro-2",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"admin-1", "admin-2"}, cfg.AuthTokens())
		assert.Equal(t, []string{"rw-1", "rw-2"}, cfg.ReadWriteAuthTokens())
		assert.Equal(t, []string{"ro-1", "ro-2"}, cfg.ReadOnlyAuthTokens())
	})

	t.Run("ignores empty auth token entries", func(t *testing.T) {
		cfg, err := Parse([]string{
			"--data-dir", t.TempDir(),
			"--auth-token", strings.Join([]string{"admin-1", "", " admin-2 ", " "}, ","),
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"admin-1", "admin-2"}, cfg.AuthTokens())
	})
}

func Test_validateTransactionTimeout(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantErr  bool
	}{
		{
			name:     "valid - 1 second",
			duration: time.Second,
			wantErr:  false,
		},
		{
			name:     "valid - 1 minute",
			duration: time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid - negative",
			duration: -time.Second,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransactionTimeout(tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
