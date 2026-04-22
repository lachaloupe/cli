package main

import "github.com/lachaloupe/cli"

var CLI = cli.Command{
	Handler: Run,
}

func Run() error {
	return nil
}
