package cli

import (
	"context"
	"io"
	"os"
)

type stdoutKey struct{}
type stderrKey struct{}
type stdinKey struct{}

// WithStdout returns a context that carries w as the standard output writer.
func WithStdout(ctx context.Context, w io.Writer) context.Context {
	if w == nil {
		panic("cli.WithStdout: nil writer")
	}

	return context.WithValue(ctx, stdoutKey{}, w)
}

// WithStderr returns a context that carries w as the standard error writer.
func WithStderr(ctx context.Context, w io.Writer) context.Context {
	if w == nil {
		panic("cli.WithStderr: nil writer")
	}

	return context.WithValue(ctx, stderrKey{}, w)
}

// WithStdin returns a context that carries r as the standard input reader.
func WithStdin(ctx context.Context, r io.Reader) context.Context {
	if r == nil {
		panic("cli.WithStdin: nil reader")
	}

	return context.WithValue(ctx, stdinKey{}, r)
}

// Stdout returns the standard output writer from ctx, or os.Stdout.
func Stdout(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stdoutKey{}).(io.Writer); ok {
		return w
	}

	return os.Stdout
}

// Stderr returns the standard error writer from ctx, or os.Stderr.
func Stderr(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stderrKey{}).(io.Writer); ok {
		return w
	}

	return os.Stderr
}

// Stdin returns the standard input reader from ctx, or os.Stdin.
func Stdin(ctx context.Context) io.Reader {
	if r, ok := ctx.Value(stdinKey{}).(io.Reader); ok {
		return r
	}

	return os.Stdin
}
