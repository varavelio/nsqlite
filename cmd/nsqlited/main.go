package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsqlite/nsqlite/internal/nsqlited"
)

func main() {
	// skip the first argument, which is the program name
	args := os.Args[1:]

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := nsqlited.Run(ctx, stop, args); err != nil {
		log.Fatal(err)
	}
}
