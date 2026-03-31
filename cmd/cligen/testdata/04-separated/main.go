package main

import (
	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name:    "login",
			Handler: RunLogin,
		},
		{
			Name:    "logout",
			Handler: RunLogout,
		},
	},
}

func main() {
	CLI.Main()
}
