package main

import (
	"context"
	"log"

	"github.com/nsqlite/nsqlite/internal/nsqlitebench"
)

// The only responsibility of the main function is to provide the operating
// system fundamentals to run nsqlitebench.

func main() {
	if err := nsqlitebench.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
