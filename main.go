package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/johnnyipcom/parseuser/cmd"
)

func main() {
	root, err := cmd.NewRoot("0.1.0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := root.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
