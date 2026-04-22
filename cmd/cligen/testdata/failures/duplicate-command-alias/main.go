package main

import (
	"context"

	"github.com/lachaloupe/cli"
)

var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name:    "publish",
			Handler: RunPublish,
		},
		{
			Name:    "push",
			Handler: RunPush,
		},
	},
}

//cli:alias=push
func RunPublish(context.Context) error {
	return nil
}

func RunPush(context.Context) error {
	return nil
}
