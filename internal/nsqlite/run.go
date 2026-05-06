package nsqlite

import (
	"context"
	"fmt"
	"io"

	"github.com/varavelio/nsqlite/internal/nsqlite/config"
	"github.com/varavelio/nsqlite/internal/nsqlite/db"
	"github.com/varavelio/nsqlite/internal/nsqlite/log"
	"github.com/varavelio/nsqlite/internal/nsqlite/server"
	"github.com/varavelio/nsqlite/internal/nsqlite/stats"
	"github.com/varavelio/nsqlite/internal/version"
)

// Run runs the NSQLite server.
func Run(ctx context.Context, stop context.CancelFunc, stdout io.Writer, args []string) error {
	conf := config.MustParse(args)

	_, _ = fmt.Fprintln(stdout, version.AsciiArt)
	logger := log.NewLogger(stdout)
	logger.Info("starting NSQLite server", log.KV{
		"dataDir":       conf.DataDir,
		"listenHost":    conf.ListenHost,
		"listenPort":    conf.ListenPort,
		"txIdleTimeout": conf.TxIdleTimeout.String(),
	})

	dbStats := stats.NewDBStats()
	defer dbStats.Close()

	dbInstance, err := db.NewDB(db.Config{
		Logger:        logger,
		DBStats:       dbStats,
		DataDir:       conf.DataDir,
		TxIdleTimeout: conf.TxIdleTimeout,
	})
	if err != nil {
		return fmt.Errorf("error starting database: %w", err)
	}
	defer func() {
		if err := dbInstance.Close(); err != nil {
			logger.Error("error closing database:", log.KV{"error": err})
		}
	}()

	serv, err := server.NewServer(server.Config{
		Logger:     logger,
		DBStats:    dbStats,
		DB:         dbInstance,
		AuthToken:  conf.AuthToken,
		ListenHost: conf.ListenHost,
		ListenPort: conf.ListenPort,
	})
	if err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}
	defer func() {
		if err := serv.Stop(); err != nil {
			logger.Error("error stopping server:", log.KV{"error": err})
		}
	}()
	go func() {
		if err := serv.Start(); err != nil {
			logger.Error("server stopped with error:", log.KV{"error": err})
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("goodbye! gracefully shutting down NSQLite server")
	return nil
}
