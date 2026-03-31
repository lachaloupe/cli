package main

import (
	"context"
	"fmt"

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

type LoginArgs struct {
	User     string
	Password string
}

func RunLogin(ctx context.Context, args LoginArgs) error {
	fmt.Println("login")
	fmt.Printf("%+v\n", args)
	return nil
}

//cli:alias=signout
func RunLogout(ctx context.Context) error {
	fmt.Println("logout")
	return nil
}

func main() {
	CLI.Main()
}
