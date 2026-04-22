package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Template: template(),
	Handler:  Run,
}

func Run(context.Context) error {
	return nil
}

func template() string {
	return "template"
}
