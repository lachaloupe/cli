package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Handler: Run,
}

type Args struct {
	//cli:alias=help
	Verbose bool
}

func Run(ctx context.Context, args Args) error {
	_, _ = ctx, args
	return nil
}
