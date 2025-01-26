package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsqlite/nsqlite/internal/nsqlitebench"
)

// The only responsibility of the main function is to provide the operating
// system fundamentals to run nsqlitebench.

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := nsqlitebench.Run(ctx, stop); err != nil {
		log.Fatal(err)
	}
}
