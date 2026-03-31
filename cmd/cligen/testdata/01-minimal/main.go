package main

import (
	"context"
	"fmt"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: Run,
}

type Args struct {
	// SomeFlag is a flag argument.
	SomeFlag string
}

// Run a command
func Run(ctx context.Context, args Args) error {
	fmt.Println(args)
	return nil
}

func main() {
	CLI.Main()
}
