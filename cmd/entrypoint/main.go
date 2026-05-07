// Package main is the container entrypoint for NSQLite.
//
// The startup flow is intentionally small:
//   - when Litestream is disabled, the container execs NSQLite directly
//   - when Litestream is enabled, the container renders a minimal Litestream
//     config for an S3-compatible replica and then execs Litestream
//
// Keeping this code in Go makes the startup rules easy to test, easy to read,
// and easy to change without shell quoting edge cases.
package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

const (
	nsqliteBinary    = "/usr/local/bin/nsqlite"
	litestreamBinary = "/usr/local/bin/litestream"

	defaultDataDir                          = "/data"
	defaultDatabaseName                     = "database.sqlite"
	defaultLitestreamConfigPath             = "/tmp/litestream.yml"
	defaultLogLevel                         = "info"
	defaultLogFormat                        = "text"
	defaultSnapshotInterval                 = "24h"
	defaultSnapshotRetention                = "168h"
	defaultSyncInterval                     = "1s"
	defaultValidationInterval               = "5m"
	configFileMode              os.FileMode = 0o600
	configDirMode               os.FileMode = 0o755
)

type environment map[string]string

// entrypointPlan describes the exact process replacement the container will perform.
type entrypointPlan struct {
	Path         string
	Args         []string
	DatabasePath string
	ConfigPath   string
	ConfigBytes  []byte
	ExtraEnv     environment
}

// litestreamSettings contains the small set of replica settings this container supports.
type litestreamSettings struct {
	Bucket             string
	Path               string
	Endpoint           string
	Region             string
	AccessKeyID        string
	SecretAccessKey    string
	SessionToken       string
	ConfigPath         string
	LogLevel           string
	LogFormat          string
	SnapshotInterval   string
	SnapshotRetention  string
	SyncInterval       string
	ValidationInterval string
}

type litestreamConfig struct {
	Logging  litestreamLogging  `yaml:"logging"`
	Snapshot litestreamSnapshot `yaml:"snapshot"`
	DBs      []litestreamDB     `yaml:"dbs"`
}

type litestreamLogging struct {
	Level string `yaml:"level"`
	Type  string `yaml:"type"`
}

type litestreamSnapshot struct {
	Interval  string `yaml:"interval"`
	Retention string `yaml:"retention"`
}

type litestreamDB struct {
	Path    string            `yaml:"path"`
	Replica litestreamReplica `yaml:"replica"`
}

type litestreamReplica struct {
	Type               string `yaml:"type"`
	Bucket             string `yaml:"bucket"`
	Path               string `yaml:"path"`
	Endpoint           string `yaml:"endpoint"`
	Region             string `yaml:"region"`
	SyncInterval       string `yaml:"sync-interval"`
	ValidationInterval string `yaml:"validation-interval"`
}

func main() {
	plan, err := buildEntrypointPlan(loadEnvironment(), os.Args[1:])
	if err != nil {
		logf("error: %v", err)
		os.Exit(1)
	}

	if len(plan.ConfigBytes) > 0 {
		if err := prepareRuntimePaths(
			plan.ConfigPath,
			plan.DatabasePath,
			plan.ConfigBytes,
		); err != nil {
			logf("error preparing Litestream runtime files: %v", err)
			os.Exit(1)
		}

		logf("starting NSQLite with Litestream replication")
		logf("database path: %s", plan.DatabasePath)
		logf("replica config: %s", plan.ConfigPath)
	}

	if err := syscall.Exec(
		plan.Path,
		plan.Args,
		mergeExecEnvironment(os.Environ(), plan.ExtraEnv),
	); err != nil {
		logf("exec failed: %v", err)
		os.Exit(1)
	}
}

// buildEntrypointPlan converts container environment and CLI args into a single exec plan.
func buildEntrypointPlan(env environment, args []string) (entrypointPlan, error) {
	nsqliteArgs := append([]string{nsqliteBinary}, args...)

	enabled, err := env.bool("NSQLITE_LITESTREAM_ENABLED", false)
	if err != nil {
		return entrypointPlan{}, fmt.Errorf("NSQLITE_LITESTREAM_ENABLED: %w", err)
	}
	if !enabled {
		return entrypointPlan{Path: nsqliteBinary, Args: nsqliteArgs, ExtraEnv: environment{}}, nil
	}

	settings, err := loadLitestreamSettings(env)
	if err != nil {
		return entrypointPlan{}, err
	}

	databasePath := resolveDatabasePath(env.defaultValue("NSQLITE_DATA_DIR", defaultDataDir))
	configBytes, err := renderLitestreamConfig(databasePath, settings)
	if err != nil {
		return entrypointPlan{}, err
	}

	return entrypointPlan{
		Path: litestreamBinary,
		Args: []string{
			litestreamBinary,
			"replicate",
			"-config",
			settings.ConfigPath,
			"-exec",
			shellJoin(nsqliteArgs),
		},
		DatabasePath: databasePath,
		ConfigPath:   settings.ConfigPath,
		ConfigBytes:  configBytes,
		ExtraEnv:     settings.execEnvironment(),
	}, nil
}

// loadLitestreamSettings reads the supported Litestream settings from the environment.
//
// The container intentionally exposes a narrow, explicit S3-compatible contract.
// When Litestream is enabled, the replica destination and credentials must be present.
func loadLitestreamSettings(env environment) (litestreamSettings, error) {
	bucket, err := env.required("NSQLITE_LITESTREAM_S3_BUCKET")
	if err != nil {
		return litestreamSettings{}, err
	}

	path, err := env.required("NSQLITE_LITESTREAM_S3_PATH")
	if err != nil {
		return litestreamSettings{}, err
	}

	endpoint, err := env.required("NSQLITE_LITESTREAM_S3_ENDPOINT")
	if err != nil {
		return litestreamSettings{}, err
	}

	region, err := env.required("NSQLITE_LITESTREAM_S3_REGION")
	if err != nil {
		return litestreamSettings{}, err
	}

	accessKeyID, err := env.required("NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID")
	if err != nil {
		return litestreamSettings{}, err
	}

	secretAccessKey, err := env.required("NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY")
	if err != nil {
		return litestreamSettings{}, err
	}

	return litestreamSettings{
		Bucket:          bucket,
		Path:            strings.TrimLeft(path, "/"),
		Endpoint:        endpoint,
		Region:          region,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    strings.TrimSpace(env["NSQLITE_LITESTREAM_S3_SESSION_TOKEN"]),
		ConfigPath: env.defaultValue(
			"NSQLITE_LITESTREAM_CONFIG_PATH",
			defaultLitestreamConfigPath,
		),
		LogLevel:  env.defaultValue("NSQLITE_LITESTREAM_LOG_LEVEL", defaultLogLevel),
		LogFormat: env.defaultValue("NSQLITE_LITESTREAM_LOG_FORMAT", defaultLogFormat),
		SnapshotInterval: env.defaultValue(
			"NSQLITE_LITESTREAM_SNAPSHOT_INTERVAL",
			defaultSnapshotInterval,
		),
		SnapshotRetention: env.defaultValue(
			"NSQLITE_LITESTREAM_SNAPSHOT_RETENTION",
			defaultSnapshotRetention,
		),
		SyncInterval: env.defaultValue(
			"NSQLITE_LITESTREAM_SYNC_INTERVAL",
			defaultSyncInterval,
		),
		ValidationInterval: env.defaultValue(
			"NSQLITE_LITESTREAM_VALIDATION_INTERVAL",
			defaultValidationInterval,
		),
	}, nil
}

// execEnvironment returns the runtime credential environment that Litestream expects.
//
// The container exposes a single public variable name for each credential, and
// translates them here into the AWS-compatible names that Litestream reads.
func (settings litestreamSettings) execEnvironment() environment {
	env := environment{
		"AWS_ACCESS_KEY_ID":     settings.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": settings.SecretAccessKey,
	}
	if settings.SessionToken != "" {
		env["AWS_SESSION_TOKEN"] = settings.SessionToken
	}
	return env
}

// renderLitestreamConfig builds the minimal Litestream YAML file used by the container.
func renderLitestreamConfig(databasePath string, settings litestreamSettings) ([]byte, error) {
	config := litestreamConfig{
		Logging: litestreamLogging{
			Level: settings.LogLevel,
			Type:  settings.LogFormat,
		},
		Snapshot: litestreamSnapshot{
			Interval:  settings.SnapshotInterval,
			Retention: settings.SnapshotRetention,
		},
		DBs: []litestreamDB{{
			Path: databasePath,
			Replica: litestreamReplica{
				Type:               "s3",
				Bucket:             settings.Bucket,
				Path:               settings.Path,
				Endpoint:           settings.Endpoint,
				Region:             settings.Region,
				SyncInterval:       settings.SyncInterval,
				ValidationInterval: settings.ValidationInterval,
			},
		}},
	}

	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal Litestream config: %w", err)
	}

	return configBytes, nil
}

func prepareRuntimePaths(configPath, databasePath string, configBytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(databasePath), configDirMode); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), configDirMode); err != nil {
		return fmt.Errorf("create Litestream config directory: %w", err)
	}

	if err := os.WriteFile(configPath, configBytes, configFileMode); err != nil {
		return fmt.Errorf("write Litestream config file: %w", err)
	}

	return nil
}

func resolveDatabasePath(dataDir string) string {
	cleanDataDir := strings.TrimRight(dataDir, "/")
	if cleanDataDir == "" {
		cleanDataDir = defaultDataDir
	}

	return filepath.Join(cleanDataDir, defaultDatabaseName)
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}

	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\n\r'\"\\$`!#&()*;<>?[]{}|~") {
		return value
	}

	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func loadEnvironment() environment {
	env := make(environment, len(os.Environ()))
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func mergeExecEnvironment(base []string, overrides environment) []string {
	merged := make(environment, len(base)+len(overrides))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			merged[key] = value
		}
	}

	maps.Copy(merged, overrides)

	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}

	return result
}

func (env environment) required(key string) (string, error) {
	value := strings.TrimSpace(env[key])
	if value == "" {
		return "", fmt.Errorf("litestream requires %s when NSQLITE_LITESTREAM_ENABLED=true", key)
	}
	return value, nil
}

func (env environment) defaultValue(key, fallback string) string {
	value := strings.TrimSpace(env[key])
	if value == "" {
		return fallback
	}
	return value
}

func (env environment) bool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(env[key])
	if value == "" {
		return fallback, nil
	}
	return parseBool(value)
}

func logf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "[entrypoint] "+format+"\n", args...)
}
