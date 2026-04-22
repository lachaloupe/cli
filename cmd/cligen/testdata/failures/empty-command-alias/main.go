package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Handler: Run,
}

//cli:alias=
func Run(context.Context) error {
	return nil
}
