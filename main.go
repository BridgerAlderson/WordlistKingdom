package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wordlistkingdom/internal/app"
	"wordlistkingdom/internal/cli"
)

// version is overridden at build time with -X main.version=...
var version = "2.0.0"

func main() {
	cli.Version = version

	opts, err := cli.Parse(os.Args[1:], os.Stderr)
	if err != nil {
		if cli.IsHelp(err) {
			return
		}
		fmt.Fprintf(os.Stderr, "[-] %v\n", err)
		fmt.Fprintln(os.Stderr, "    run 'wordlistkingdom -h' for usage")
		os.Exit(2)
	}
	if opts.Version {
		fmt.Printf("WordlistKingdom %s\n", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if _, err := app.Run(ctx, opts.Config, version, os.Stdout, os.Stderr); err != nil {
		switch {
		case errors.Is(err, app.ErrInterrupted):
			fmt.Fprintln(os.Stderr, "[!] Interrupted; partial wordlist flushed.")
			os.Exit(130)
		case errors.Is(err, app.ErrNoCandidates):
			fmt.Fprintln(os.Stderr, "[-] No candidates survived filtering.")
			os.Exit(1)
		default:
			fmt.Fprintf(os.Stderr, "[-] %v\n", err)
			os.Exit(1)
		}
	}
}
