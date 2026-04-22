package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Handler: Run,
	Commands: []*cli.Command{
		{
			Name:    "push",
			Handler: RunPush,
		},
	},
}

type Args struct {
	//cli:arg
	Paths []string
}

func Run(ctx context.Context, args Args) error {
	_, _ = ctx, args
	return nil
}

func RunPush(context.Context) error {
	return nil
}
