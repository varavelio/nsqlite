package main

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr string
	}{
		{name: "true", value: "true", want: true},
		{name: "uppercase true", value: "YES", want: true},
		{name: "false", value: "false", want: false},
		{name: "numeric false", value: "0", want: false},
		{name: "invalid", value: "sometimes", wantErr: "invalid boolean value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBool(tt.value)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBuildEntrypointPlan(t *testing.T) {
	t.Run("execs nsqlite directly when litestream is disabled", func(t *testing.T) {
		plan, err := buildEntrypointPlan(environment{
			"NSQLITE_LITESTREAM_ENABLED": "false",
		}, []string{"--listen-port", "9876"})

		require.NoError(t, err)
		require.Equal(t, entrypointPlan{
			Path:     nsqliteBinary,
			Args:     []string{nsqliteBinary, "--listen-port", "9876"},
			ExtraEnv: environment{},
		}, plan)
	})

	t.Run("builds a litestream plan from explicit s3 settings", func(t *testing.T) {
		plan, err := buildEntrypointPlan(environment{
			"NSQLITE_LITESTREAM_ENABLED":              "true",
			"NSQLITE_DATA_DIR":                        "/var/lib/nsqlite",
			"NSQLITE_LITESTREAM_S3_BUCKET":            "backups",
			"NSQLITE_LITESTREAM_S3_PATH":              "prod/database.sqlite",
			"NSQLITE_LITESTREAM_S3_ENDPOINT":          "https://minio.example.com:9000",
			"NSQLITE_LITESTREAM_S3_REGION":            "us-east-1",
			"NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID":     "access-key",
			"NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY": "secret-key",
			"NSQLITE_LITESTREAM_LOG_LEVEL":            "debug",
		}, []string{"--listen-port", "9999", "--auth-token", "hello world"})

		require.NoError(t, err)
		require.Equal(t, litestreamBinary, plan.Path)
		require.Equal(t, "/var/lib/nsqlite/database.sqlite", plan.DatabasePath)
		require.Equal(t, defaultLitestreamConfigPath, plan.ConfigPath)
		require.Equal(t, environment{
			"AWS_ACCESS_KEY_ID":     "access-key",
			"AWS_SECRET_ACCESS_KEY": "secret-key",
		}, plan.ExtraEnv)
		require.Equal(t, []string{
			litestreamBinary,
			"replicate",
			"-config",
			defaultLitestreamConfigPath,
			"-exec",
			"/usr/local/bin/nsqlite --listen-port 9999 --auth-token 'hello world'",
		}, plan.Args)

		var config litestreamConfig
		require.NoError(t, yaml.Unmarshal(plan.ConfigBytes, &config))
		require.Equal(t, "debug", config.Logging.Level)
		require.Equal(t, defaultLogFormat, config.Logging.Type)
		require.Equal(t, defaultSnapshotInterval, config.Snapshot.Interval)
		require.Equal(t, defaultSnapshotRetention, config.Snapshot.Retention)
		require.Len(t, config.DBs, 1)
		require.Equal(t, "/var/lib/nsqlite/database.sqlite", config.DBs[0].Path)
		require.Equal(t, "s3", config.DBs[0].Replica.Type)
		require.Equal(t, "backups", config.DBs[0].Replica.Bucket)
		require.Equal(t, "prod/database.sqlite", config.DBs[0].Replica.Path)
		require.Equal(t, "https://minio.example.com:9000", config.DBs[0].Replica.Endpoint)
		require.Equal(t, "us-east-1", config.DBs[0].Replica.Region)
		require.Equal(t, defaultSyncInterval, config.DBs[0].Replica.SyncInterval)
		require.Equal(t, defaultValidationInterval, config.DBs[0].Replica.ValidationInterval)
		require.NotContains(t, string(plan.ConfigBytes), "access-key")
		require.NotContains(t, string(plan.ConfigBytes), "secret-key")
	})

	t.Run("passes optional session token through translated runtime env", func(t *testing.T) {
		plan, err := buildEntrypointPlan(environment{
			"NSQLITE_LITESTREAM_ENABLED":              "true",
			"NSQLITE_LITESTREAM_S3_BUCKET":            "backups",
			"NSQLITE_LITESTREAM_S3_PATH":              "prod/database.sqlite",
			"NSQLITE_LITESTREAM_S3_ENDPOINT":          "minio.example.com:9000",
			"NSQLITE_LITESTREAM_S3_REGION":            "us-east-1",
			"NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID":     "access-key",
			"NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY": "secret-key",
			"NSQLITE_LITESTREAM_S3_SESSION_TOKEN":     "session-token",
			"NSQLITE_LITESTREAM_CONFIG_PATH":          "/tmp/custom-litestream.yml",
		}, nil)

		require.NoError(t, err)
		require.Equal(t, environment{
			"AWS_ACCESS_KEY_ID":     "access-key",
			"AWS_SECRET_ACCESS_KEY": "secret-key",
			"AWS_SESSION_TOKEN":     "session-token",
		}, plan.ExtraEnv)
	})

	t.Run("fails clearly when litestream enabled is invalid", func(t *testing.T) {
		_, err := buildEntrypointPlan(environment{
			"NSQLITE_LITESTREAM_ENABLED": "maybe",
		}, nil)

		require.Error(t, err)
		require.ErrorContains(t, err, "NSQLITE_LITESTREAM_ENABLED")
		require.ErrorContains(t, err, "invalid boolean value")
	})

	t.Run("requires every mandatory litestream s3 setting", func(t *testing.T) {
		baseEnv := environment{
			"NSQLITE_LITESTREAM_ENABLED":              "true",
			"NSQLITE_LITESTREAM_S3_BUCKET":            "backups",
			"NSQLITE_LITESTREAM_S3_PATH":              "prod/database.sqlite",
			"NSQLITE_LITESTREAM_S3_ENDPOINT":          "minio.example.com:9000",
			"NSQLITE_LITESTREAM_S3_REGION":            "us-east-1",
			"NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID":     "access-key",
			"NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY": "secret-key",
		}

		tests := []struct {
			name       string
			removeKey  string
			wantErrKey string
		}{
			{
				name:       "missing bucket",
				removeKey:  "NSQLITE_LITESTREAM_S3_BUCKET",
				wantErrKey: "NSQLITE_LITESTREAM_S3_BUCKET",
			},
			{
				name:       "missing path",
				removeKey:  "NSQLITE_LITESTREAM_S3_PATH",
				wantErrKey: "NSQLITE_LITESTREAM_S3_PATH",
			},
			{
				name:       "missing endpoint",
				removeKey:  "NSQLITE_LITESTREAM_S3_ENDPOINT",
				wantErrKey: "NSQLITE_LITESTREAM_S3_ENDPOINT",
			},
			{
				name:       "missing region",
				removeKey:  "NSQLITE_LITESTREAM_S3_REGION",
				wantErrKey: "NSQLITE_LITESTREAM_S3_REGION",
			},
			{
				name:       "missing access key",
				removeKey:  "NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID",
				wantErrKey: "NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID",
			},
			{
				name:       "missing secret key",
				removeKey:  "NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY",
				wantErrKey: "NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := cloneEnvironment(baseEnv)
				delete(env, tt.removeKey)

				_, err := buildEntrypointPlan(env, nil)
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrKey)
			})
		}
	})

	t.Run("quotes nsqlite args for litestream exec", func(t *testing.T) {
		plan, err := buildEntrypointPlan(environment{
			"NSQLITE_LITESTREAM_ENABLED":              "true",
			"NSQLITE_LITESTREAM_S3_BUCKET":            "backups",
			"NSQLITE_LITESTREAM_S3_PATH":              "prod/database.sqlite",
			"NSQLITE_LITESTREAM_S3_ENDPOINT":          "minio.example.com:9000",
			"NSQLITE_LITESTREAM_S3_REGION":            "us-east-1",
			"NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID":     "access-key",
			"NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY": "secret-key",
		}, []string{"--auth-token", "hello world", "--data-dir=/tmp/it's"})

		require.NoError(t, err)
		require.Equal(
			t,
			"/usr/local/bin/nsqlite --auth-token 'hello world' '--data-dir=/tmp/it'\\''s'",
			plan.Args[5],
		)
	})
}

func TestPrepareRuntimePaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config", "litestream.yml")
	databasePath := filepath.Join(dir, "db", "database.sqlite")

	err := prepareRuntimePaths(configPath, databasePath, []byte("logging: {}\n"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Dir(databasePath))
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func cloneEnvironment(env environment) environment {
	cloned := make(environment, len(env))
	maps.Copy(cloned, env)
	return cloned
}
