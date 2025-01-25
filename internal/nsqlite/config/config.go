package config

import (
	"fmt"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/nsqlite/nsqlite/internal/version"
	"github.com/nsqlite/nsqlitego/nsqlitedsn"
)

// Config represents the configuration for nsqlite.
type Config struct {
	ConnectionString string              `arg:"positional" help:"Connection string for the NSQLite database server in format http(s)://host:port?authToken=value (default to http://localhost:9876)" default:"http://localhost:9876"`
	ParsedConnStr    *nsqlitedsn.ConnStr `arg:"-"`
}

func (Config) Version() string {
	return fmt.Sprintf("%s\n", version.CLIVersion())
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

	cfg.ParsedConnStr, err = nsqlitedsn.NewConnStrFromText(cfg.ConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}
