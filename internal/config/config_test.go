package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("allows running without auth tokens", func(t *testing.T) {
		cfg, err := Parse([]string{"--data-dir", t.TempDir()})
		require.NoError(t, err)
		assert.Empty(t, cfg.AuthTokens())
		assert.Empty(t, cfg.ReadWriteAuthTokens())
		assert.Empty(t, cfg.ReadOnlyAuthTokens())
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

	t.Run("parses multiple tokens when argon2 hashes are present", func(t *testing.T) {
		const (
			adminArgon2 = "$argon2id$v=19$m=16,t=2,p=1$YWRtaW4tc2FsdA$YWRtaW4taGFzaA"
			rwArgon2    = "$argon2id$v=19$m=32,t=3,p=2$cnctc2FsdA$cnctaGFzaA"
			roArgon2    = "$argon2id$v=19$m=64,t=4,p=3$cm8tc2FsdA$cm8taGFzaA"
		)

		cfg, err := Parse([]string{
			"--data-dir", t.TempDir(),
			"--auth-token", strings.Join([]string{"admin-1", adminArgon2, "admin-2"}, ","),
			"--auth-token-rw", strings.Join([]string{"rw-1", rwArgon2, "rw-2"}, ","),
			"--auth-token-ro", strings.Join([]string{"ro-1", roArgon2, "ro-2"}, ","),
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"admin-1", adminArgon2, "admin-2"}, cfg.AuthTokens())
		assert.Equal(t, []string{"rw-1", rwArgon2, "rw-2"}, cfg.ReadWriteAuthTokens())
		assert.Equal(t, []string{"ro-1", roArgon2, "ro-2"}, cfg.ReadOnlyAuthTokens())
	})
}

func TestSplitAuthTokens(t *testing.T) {
	t.Run("returns nil for empty input", func(t *testing.T) {
		assert.Nil(t, splitAuthTokens(""))
	})

	t.Run("splits tokens only on commas", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "rw", "ro"}, splitAuthTokens("admin,rw,ro"))
	})

	t.Run("trims whitespace around each token", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "rw", "ro"}, splitAuthTokens(" admin , rw , ro "))
	})

	t.Run("preserves spaces inside tokens", func(t *testing.T) {
		assert.Equal(
			t,
			[]string{"admin token", "rw token"},
			splitAuthTokens("admin token,rw token"),
		)
	})

	t.Run("does not split on spaces", func(t *testing.T) {
		assert.Equal(t, []string{"admin rw ro"}, splitAuthTokens("admin rw ro"))
	})

	t.Run("does not split on tabs", func(t *testing.T) {
		assert.Equal(t, []string{"admin\trw"}, splitAuthTokens("admin\trw"))
	})

	t.Run("does not split on newlines", func(t *testing.T) {
		assert.Equal(t, []string{"admin\nrw"}, splitAuthTokens("admin\nrw"))
	})

	t.Run("ignores empty entries created by commas", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "rw"}, splitAuthTokens(",admin,,rw,"))
	})

	t.Run("trims tabs and newlines only at token edges", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "rw"}, splitAuthTokens("\nadmin\t,\trw\n"))
	})

	t.Run("does not split inside argon2 parameter lists", func(t *testing.T) {
		const (
			argon2A = "$argon2id$v=19$m=16,t=2,p=1$YWJjZA$ZWZnaA"
			argon2B = "$argon2id$v=19$m=32,t=3,p=2$aWprbA$bW5vcA"
		)

		assert.Equal(
			t,
			[]string{"admin", argon2A, argon2B, "rw"},
			splitAuthTokens(strings.Join([]string{"admin", argon2A, argon2B, "rw"}, ",")),
		)
	})

	t.Run("does not swallow following tokens after a truncated argon2 hash", func(t *testing.T) {
		assert.Equal(
			t,
			[]string{"admin", "$argon2id$v=19$m=16,t=2,p=1", "rw"},
			splitAuthTokens("admin,$argon2id$v=19$m=16,t=2,p=1,rw"),
		)
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
