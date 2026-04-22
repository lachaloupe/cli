package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Handler: Run,
}

type Args struct {
	//cli:arg=nope
	Path string
}

func Run(ctx context.Context, args Args) error {
	_, _ = ctx, args
	return nil
}
