package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/varavelio/nsqlite/internal/validate"
	"github.com/varavelio/nsqlite/internal/version"
)

// Config represents the configuration for nsqlited.
type Config struct {
	AuthToken     string        `arg:"--auth-token,env:NSQLITE_AUTH_TOKEN"           help:"Admin authentication token(s). Use comma-separated plaintext or bcrypt/argon2 hashes for full access."`
	AuthTokenRW   string        `arg:"--auth-token-rw,env:NSQLITE_AUTH_TOKEN_RW"     help:"Read/write authentication token(s). Use comma-separated plaintext or bcrypt/argon2 hashes for query read/write access only."`
	AuthTokenRO   string        `arg:"--auth-token-ro,env:NSQLITE_AUTH_TOKEN_RO"     help:"Read-only authentication token(s). Use comma-separated plaintext or bcrypt/argon2 hashes for query read access only."`
	DataDir       string        `arg:"--data-dir,env:NSQLITE_DATA_DIR"               help:"Directory for NSQLite database files"                                                                                        default:"./data"`
	ListenHost    string        `arg:"--listen-host,env:NSQLITE_LISTEN_HOST"         help:"Host for the server to listen on"                                                                                            default:"0.0.0.0"`
	ListenPort    string        `arg:"--listen-port,env:NSQLITE_LISTEN_PORT"         help:"Port for the server to listen on"                                                                                            default:"9876"`
	TxIdleTimeout time.Duration `arg:"--tx-idle-timeout,env:NSQLITE_TX_IDLE_TIMEOUT" help:"If a transaction is not active for this duration, it will be rolled back. Valid time units are ns, us (or µs), ms, s, m, h"  default:"10s"`
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
	if c.TxIdleTimeout != time.Duration(0) {
		args = append(args, "--tx-idle-timeout", c.TxIdleTimeout.String())
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

// splitAuthTokens splits a comma-separated token list and trims each token.
func splitAuthTokens(value string) []string {
	if value == "" {
		return nil
	}

	tokens := make([]string, 0, strings.Count(value, ",")+1)
	start := 0
	inArgon2 := false
	checkedTokenPrefix := false
	argon2DollarCount := 0
	argon2IgnoredCommas := 0

	appendToken := func(end int) {
		token := strings.TrimSpace(value[start:end])
		if token != "" {
			tokens = append(tokens, token)
		}
	}

	for i := range len(value) {
		if !checkedTokenPrefix {
			switch value[i] {
			case ' ', '\t', '\n', '\r':
				continue
			case ',':
				start = i + 1
				continue
			default:
				inArgon2 = strings.HasPrefix(value[i:], "$argon2id$")
				checkedTokenPrefix = true
			}
		}

		switch value[i] {
		case '$':
			if inArgon2 {
				argon2DollarCount++
			}
		case ',':
			if inArgon2 && argon2DollarCount == 3 && argon2IgnoredCommas < 2 {
				argon2IgnoredCommas++
				continue
			}

			appendToken(i)
			start = i + 1
			inArgon2 = false
			checkedTokenPrefix = false
			argon2DollarCount = 0
			argon2IgnoredCommas = 0
		}
	}

	appendToken(len(value))

	return tokens
}
