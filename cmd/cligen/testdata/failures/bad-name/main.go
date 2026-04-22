package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Name:    "$",
	Handler: Run,
}

func Run(context.Context) error {
	return nil
}
