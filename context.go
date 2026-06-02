package cli

import (
	"context"
	"io"
	"os"
)

type stdoutKey struct{}
type stderrKey struct{}
type stdinKey struct{}

func WithStdout(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, stdoutKey{}, w)
}

func WithStderr(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, stderrKey{}, w)
}

func WithStdin(ctx context.Context, r io.Reader) context.Context {
	return context.WithValue(ctx, stdinKey{}, r)
}

func Stdout(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stdoutKey{}).(io.Writer); ok {
		return w
	}

	return os.Stdout
}

func Stderr(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stderrKey{}).(io.Writer); ok {
		return w
	}

	return os.Stderr
}

func Stdin(ctx context.Context) io.Reader {
	if r, ok := ctx.Value(stdinKey{}).(io.Reader); ok {
		return r
	}

	return os.Stdin
}
