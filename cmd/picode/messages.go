package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cfpperche/picode/internal/communication"
)

func runMessages(args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := communication.RunCLI(ctx, args, os.Stdin, os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "messages:", err)
		os.Exit(1)
	}
}
