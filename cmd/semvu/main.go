// Command semvu (Semantic Version Utilities) reads and modifies semantic version strings.
//
// This file is the process boundary: it collects the real arguments, streams
// and signals, hands them to the cli package, and exits with the code it
// returns. Keep it free of logic so that everything else stays testable.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/tedslittlerobot/semvu/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	code := cli.Execute(ctx, os.Args[1:], cli.Streams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	})

	stop()
	os.Exit(code)
}
