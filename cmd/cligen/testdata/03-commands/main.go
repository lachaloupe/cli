package main

import (
	"context"
	"log/slog"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Context: func(ctx context.Context) context.Context {
		h := slog.NewTextHandler(cli.Stderr(ctx), &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		})
		return WithLogger(ctx, slog.New(h))
	},
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

type loggerKey struct{}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

func Logger(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey{}).(*slog.Logger)
}

type LoginArgs struct {
	User     string
	Password string
}

func RunLogin(ctx context.Context, args LoginArgs) error {
	Logger(ctx).Info("login", "user", args.User, "password", args.Password)
	return nil
}

//cli:alias=signout
func RunLogout(ctx context.Context) error {
	Logger(ctx).Info("logout")
	return nil
}

func main() {
	CLI.Main()
}
