package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/varavelio/nsqlite/internal/config"
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/logger"
	"github.com/varavelio/nsqlite/internal/server"
	"github.com/varavelio/nsqlite/internal/stats"
	"github.com/varavelio/nsqlite/internal/version"
)

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, stop, os.Stdout, os.Args[1:]); err != nil {
		msg := fmt.Sprintf(
			"%s %s: %s",
			time.Now().UTC().Format(time.RFC3339),
			"error running NSQLite server",
			err.Error(),
		)
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

func run(ctx context.Context, stop context.CancelFunc, stdout io.Writer, args []string) error {
	conf := config.MustParse(args)

	_, _ = fmt.Fprintln(stdout, version.AsciiArt)
	log := logger.NewLogger()
	log.Info(ctx, "starting NSQLite server",
		"dataDir", conf.DataDir,
		"listenHost", conf.ListenHost,
		"listenPort", conf.ListenPort,
		"txIdleTimeout", conf.TxIdleTimeout.String(),
		"maxReadConns", conf.MaxReadConns,
	)

	dbStats := stats.NewDBStats()
	defer dbStats.Close()

	dbInstance, err := db.NewDB(db.Config{
		Logger:        log,
		DBStats:       dbStats,
		DataDir:       conf.DataDir,
		TxIdleTimeout: conf.TxIdleTimeout,
		MaxReadConns:  conf.MaxReadConns,
	})
	if err != nil {
		return fmt.Errorf("error starting database: %w", err)
	}
	defer func() {
		if err := dbInstance.Close(); err != nil {
			log.Error(ctx, "error closing database", "error", err)
		}
	}()

	serv, err := server.NewServer(server.Config{
		Logger:              log,
		DBStats:             dbStats,
		DB:                  dbInstance,
		AuthTokens:          conf.AuthTokens(),
		ReadWriteAuthTokens: conf.ReadWriteAuthTokens(),
		ReadOnlyAuthTokens:  conf.ReadOnlyAuthTokens(),
		ListenHost:          conf.ListenHost,
		ListenPort:          conf.ListenPort,
	})
	if err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}
	defer func() {
		if err := serv.Stop(); err != nil {
			log.Error(ctx, "error stopping server", "error", err)
		}
	}()

	go func() {
		if err := serv.Start(); err != nil {
			log.Error(ctx, "server stopped with error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info(ctx, "goodbye! gracefully shutting down NSQLite server")
	return nil
}
