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
	fmt.Printf("login user=%s password=%s\n", args.User, args.Password)
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
