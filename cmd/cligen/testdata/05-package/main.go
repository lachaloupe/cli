package main

import (
	"example.com/testcli/auth"
	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name:    "login",
			Handler: auth.RunLogin,
		},
		{
			Name:    "logout",
			Handler: auth.RunLogout,
		},
	},
}

func main() {
	CLI.Main()
}
