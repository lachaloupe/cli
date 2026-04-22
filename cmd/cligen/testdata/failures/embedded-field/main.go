package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Handler: Run,
}

type Base struct {
	Verbose bool
}

type Args struct {
	Base
}

func Run(ctx context.Context, args Args) error {
	_, _ = ctx, args
	return nil
}
