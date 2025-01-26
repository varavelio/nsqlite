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
	// skip the first argument, which is the program name
	args := os.Args[1:]

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := nsqlitebench.Run(ctx, stop, args); err != nil {
		log.Fatal(err)
	}
}
