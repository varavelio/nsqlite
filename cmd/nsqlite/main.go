package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/varavelio/nsqlite/internal/nsqlite"
)

func main() {
	// skip the first argument, which is the program name
	args := os.Args[1:]

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := nsqlite.Run(ctx, stop, os.Stdout, args); err != nil {
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
