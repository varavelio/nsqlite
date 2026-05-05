package config

import (
	"fmt"
	"log"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/nsqlite/nsqlite/internal/validate"
	"github.com/nsqlite/nsqlite/internal/version"
)

// Config represents the configuration for nsqlited.
type Config struct {
	DataDir       string        `arg:"--data-dir,env:NSQLITE_DATA_DIR"               help:"Directory for NSQLite database files"                                                                                       default:"./data"`
	AuthToken     string        `arg:"--auth-token,env:NSQLITE_AUTH_TOKEN"           help:"Authentication token (plaintext or hashed with bcrypt/argon2); leave empty to disable."`
	ListenHost    string        `arg:"--listen-host,env:NSQLITE_LISTEN_HOST"         help:"Host for the server to listen on"                                                                                           default:"0.0.0.0"`
	ListenPort    string        `arg:"--listen-port,env:NSQLITE_LISTEN_PORT"         help:"Port for the server to listen on"                                                                                           default:"9876"`
	TxIdleTimeout time.Duration `arg:"--tx-idle-timeout,env:NSQLITE_TX_IDLE_TIMEOUT" help:"If a transaction is not active for this duration, it will be rolled back. Valid time units are ns, us (or µs), ms, s, m, h" default:"10s"`
}

func (Config) Version() string {
	return fmt.Sprintf("%s\n", version.ServerVersion())
}

func (c Config) ToArgs() []string {
	args := []string{}

	if c.DataDir != "" {
		args = append(args, "--data-dir", c.DataDir)
	}
	if c.AuthToken != "" {
		args = append(args, "--auth-token", c.AuthToken)
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

// MustParse parses and validates the configuration from the command
// line arguments. It returns a Config struct or exits the program
// with an error.
func MustParse(args []string) Config {
	cfg := Config{}

	parser, err := arg.NewParser(
		arg.Config{},
		&cfg,
	)
	if err != nil {
		log.Fatal(err)
	}
	parser.MustParse(args)

	if !validate.ListenHost(cfg.ListenHost) {
		log.Fatalf("invalid listen address %s", cfg.ListenHost)
	}

	if !validate.Port(cfg.ListenPort) {
		log.Fatalf("invalid listen port %s, valid values are 1-65535", cfg.ListenPort)
	}

	if err := validateTransactionTimeout(cfg.TxIdleTimeout); err != nil {
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
