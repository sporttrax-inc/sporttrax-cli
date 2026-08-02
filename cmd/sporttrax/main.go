package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
	"github.com/sporttrax-inc/sporttrax-cli/internal/cli"
)

// Exit codes follow gh's convention and are part of the CLI's contract:
// 0 success, 1 error, 2 cancelled (Ctrl-C), 4 authentication failure.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := cli.Execute(ctx)
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, context.Canceled):
		fmt.Fprintln(os.Stderr, "cancelled")
		os.Exit(2)
	case api.IsAuthError(err), errors.Is(err, auth.ErrNotFound):
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(4)
	default:
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
