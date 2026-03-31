package auth

import (
	"context"
	"fmt"
)

type LoginArgs struct {
	User     string
	Password string
}

func RunLogin(ctx context.Context, args LoginArgs) error {
	fmt.Println("login")
	fmt.Printf("%+v\n", args)
	return nil
}

func RunLogout(ctx context.Context) error {
	fmt.Println("logout")
	return nil
}
