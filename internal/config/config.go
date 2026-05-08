package config

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/varavelio/nsqlite/internal/validate"
	"github.com/varavelio/nsqlite/internal/version"
)

// Config represents the configuration for nsqlited.
type Config struct {
	AuthToken             string        `arg:"--auth-token,env:NSQLITE_AUTH_TOKEN"                         help:"Admin authentication token(s). Use space-separated plaintext or bcrypt/argon2 hashes for full access."`
	AuthTokenRW           string        `arg:"--auth-token-rw,env:NSQLITE_AUTH_TOKEN_RW"                   help:"Read/write authentication token(s). Use space-separated plaintext or bcrypt/argon2 hashes for query read/write access only."`
	AuthTokenRO           string        `arg:"--auth-token-ro,env:NSQLITE_AUTH_TOKEN_RO"                   help:"Read-only authentication token(s). Use space-separated plaintext or bcrypt/argon2 hashes for query read access only."`
	DataDir               string        `arg:"--data-dir,env:NSQLITE_DATA_DIR"                             help:"Directory for NSQLite database files"                                                                                        default:"/data"`
	ListenHost            string        `arg:"--listen-host,env:NSQLITE_LISTEN_HOST"                       help:"Host for the server to listen on"                                                                                            default:"0.0.0.0"`
	ListenPort            string        `arg:"--listen-port,env:NSQLITE_LISTEN_PORT"                       help:"Port for the server to listen on"                                                                                            default:"9876"`
	DisableCORS           bool          `arg:"--disable-cors,env:NSQLITE_DISABLE_CORS"                     help:"Disable CORS response headers and preflight handling."`
	CORSAllowedOriginsRaw string        `arg:"--cors-allowed-origins,env:NSQLITE_CORS_ALLOWED_ORIGINS"     help:"Comma-separated list of allowed CORS origins. Use * to allow any origin when credentials are disabled."                      default:"*"`
	CORSAllowedHeadersRaw string        `arg:"--cors-allowed-headers,env:NSQLITE_CORS_ALLOWED_HEADERS"     help:"Comma-separated list of allowed CORS request headers. Use * to allow any requested header."                                  default:"Accept,Authorization,Content-Type"`
	CORSAllowCredentials  bool          `arg:"--cors-allow-credentials,env:NSQLITE_CORS_ALLOW_CREDENTIALS" help:"Allow browsers to include credentials on cross-origin requests. Requires explicit origins."`
	TxIdleTimeout         time.Duration `arg:"--tx-idle-timeout,env:NSQLITE_TX_IDLE_TIMEOUT"               help:"If a transaction is not active for this duration, it will be rolled back. Valid time units are ns, us (or µs), ms, s, m, h"  default:"10s"`
	MaxReadConns          int           `arg:"--max-read-conns,env:NSQLITE_MAX_READ_CONNS"                 help:"Maximum number of read-only database connections"                                                                            default:"10"`
	CacheSizeKB           int           `arg:"--cache-size-kb,env:NSQLITE_CACHE_SIZE_KB"                   help:"SQLite cache size in KB per connection (negative is converted internally, just specify the positive KB value)"               default:"20000"`
	BusyTimeout           time.Duration `arg:"--busy-timeout,env:NSQLITE_BUSY_TIMEOUT"                     help:"How long SQLite waits when the database is locked by another writer. Valid time units are ns, us (or µs), ms, s, m, h"       default:"5s"`
	MaxRequestSizeMB      int           `arg:"--max-request-size-mb,env:NSQLITE_MAX_REQUEST_SIZE_MB"       help:"Maximum HTTP request body size in MB for the /query endpoint"                                                                default:"100"`
}

// Version returns the CLI version banner.
func (Config) Version() string {
	return fmt.Sprintf("%s\n", version.AsciiArt)
}

// ToArgs converts the config into CLI arguments.
func (c Config) ToArgs() []string {
	args := []string{}

	if c.AuthToken != "" {
		args = append(args, "--auth-token", c.AuthToken)
	}
	if c.AuthTokenRW != "" {
		args = append(args, "--auth-token-rw", c.AuthTokenRW)
	}
	if c.AuthTokenRO != "" {
		args = append(args, "--auth-token-ro", c.AuthTokenRO)
	}
	if c.DataDir != "" {
		args = append(args, "--data-dir", c.DataDir)
	}
	if c.ListenHost != "" {
		args = append(args, "--listen-host", c.ListenHost)
	}
	if c.ListenPort != "" {
		args = append(args, "--listen-port", c.ListenPort)
	}
	if c.DisableCORS {
		args = append(args, "--disable-cors")
	}
	if c.CORSAllowedOriginsRaw != "" {
		args = append(args, "--cors-allowed-origins", c.CORSAllowedOriginsRaw)
	}
	if c.CORSAllowedHeadersRaw != "" {
		args = append(args, "--cors-allowed-headers", c.CORSAllowedHeadersRaw)
	}
	if c.CORSAllowCredentials {
		args = append(args, "--cors-allow-credentials")
	}
	if c.TxIdleTimeout != time.Duration(0) {
		args = append(args, "--tx-idle-timeout", c.TxIdleTimeout.String())
	}
	if c.MaxReadConns != 0 {
		args = append(args, "--max-read-conns", fmt.Sprintf("%d", c.MaxReadConns))
	}
	if c.CacheSizeKB != 0 {
		args = append(args, "--cache-size-kb", fmt.Sprintf("%d", c.CacheSizeKB))
	}
	if c.BusyTimeout != time.Duration(0) {
		args = append(args, "--busy-timeout", c.BusyTimeout.String())
	}
	if c.MaxRequestSizeMB != 0 {
		args = append(args, "--max-request-size-mb", fmt.Sprintf("%d", c.MaxRequestSizeMB))
	}

	return args
}

// Parse parses and validates the configuration from the command line arguments.
func Parse(args []string) (Config, error) {
	cfg := Config{}

	parser, err := arg.NewParser(
		arg.Config{},
		&cfg,
	)
	if err != nil {
		return Config{}, err
	}
	if err := parser.Parse(args); err != nil {
		return Config{}, err
	}

	if !validate.ListenHost(cfg.ListenHost) {
		return Config{}, fmt.Errorf("invalid listen address %s", cfg.ListenHost)
	}

	if !validate.Port(cfg.ListenPort) {
		return Config{}, fmt.Errorf(
			"invalid listen port %s, valid values are 1-65535",
			cfg.ListenPort,
		)
	}

	if err := validateTransactionTimeout(cfg.TxIdleTimeout); err != nil {
		return Config{}, err
	}

	if err := validateBusyTimeout(cfg.BusyTimeout); err != nil {
		return Config{}, err
	}

	if err := validateCacheSize(cfg.CacheSizeKB); err != nil {
		return Config{}, err
	}

	if err := validateCORS(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// MustParse parses and validates the configuration from the command
// line arguments. It returns a Config struct or exits the program
// with an error.
func MustParse(args []string) Config {
	cfg, err := Parse(args)
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

// validateTransactionTimeout validates if timeout is greater than zero.
func validateTransactionTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf(
			"invalid transaction timeout %s, must be greater than zero",
			timeout.String(),
		)
	}
	return nil
}

// AuthTokens returns the configured admin authentication tokens.
func (c Config) AuthTokens() []string {
	return splitAuthTokens(c.AuthToken)
}

// ReadWriteAuthTokens returns the configured read/write authentication tokens.
func (c Config) ReadWriteAuthTokens() []string {
	return splitAuthTokens(c.AuthTokenRW)
}

// ReadOnlyAuthTokens returns the configured read-only authentication tokens.
func (c Config) ReadOnlyAuthTokens() []string {
	return splitAuthTokens(c.AuthTokenRO)
}

// CORSAllowedOrigins returns the configured CORS origins.
func (c Config) CORSAllowedOrigins() []string {
	return splitCommaSeparated(c.CORSAllowedOriginsRaw)
}

// CORSAllowedHeaders returns the configured CORS request headers.
func (c Config) CORSAllowedHeaders() []string {
	return splitCommaSeparated(c.CORSAllowedHeadersRaw)
}

// validateBusyTimeout validates that the busy timeout is greater than zero.
func validateBusyTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf(
			"invalid busy timeout %s, must be greater than zero",
			timeout.String(),
		)
	}
	return nil
}

// validateCacheSize validates that the cache size is greater than zero.
func validateCacheSize(size int) error {
	if size <= 0 {
		return fmt.Errorf(
			"invalid cache size %d KB, must be greater than zero",
			size,
		)
	}
	return nil
}

func validateCORS(cfg Config) error {
	if cfg.DisableCORS {
		return nil
	}

	allowedOrigins := cfg.CORSAllowedOrigins()
	if len(allowedOrigins) == 0 {
		return fmt.Errorf("cors allowed origins must contain at least one origin")
	}

	if cfg.CORSAllowCredentials && slices.Contains(allowedOrigins, "*") {
		return fmt.Errorf("cors allowed origins cannot contain * when credentials are enabled")
	}

	return nil
}

// splitAuthTokens splits a token list on whitespace.
func splitAuthTokens(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
