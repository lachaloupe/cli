package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"syscall"
)

// Register installs the generated invocation function for the command.
func (c *Command) Register(f func(ctx context.Context, args []string) ([]*Command, error)) {
	c.invoke = f
}

// Get returns the argument that matches the provided name or alias.
func (c *Command) Get(name string) *Arg {
	for i := range c.Args {
		if arg := c.Args[i]; arg.Name == name {
			return arg
		}

		if slices.Contains(c.Args[i].Aliases, name) {
			return c.Args[i]
		}
	}

	return nil
}

// CommandList returns a flattened list of all commands in the tree rooted at c.
func (c *Command) CommandList() []*Command {
	list := []*Command{c}

	for _, cmd := range c.Commands {
		list = append(list, cmd.CommandList()...)
	}

	return list
}

// RunVersion prints the configured CLI version.
func RunVersion(ctx context.Context) error {
	w := Stdout(ctx)
	fmt.Fprintf(w, "%s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintln(w, Version)
	return nil
}

// Run executes the command tree against the provided command-line arguments.
func (c *Command) Run(ctx context.Context, args []string) ([]*Command, error) {
	f := c.invoke

	if f == nil {
		panic(fmt.Sprintf("expected cli.Register to be called for %s", c.Name))
	}

	return f(ctx, args)
}

// Main runs the command with process arguments and handles help and error output.
func (c *Command) Main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if c.Context != nil {
		ctx = c.Context(ctx)
	}

	cmds, err := c.Run(ctx, os.Args[1:])
	if err != nil {
		// Close resources opened during Parse if the handler never gets called.
		for _, cmd := range cmds {
			cmd.Cleanup()
		}

		if err != ErrHelp {
			fmt.Fprintf(Stderr(ctx), "error: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprint(Stderr(ctx), Help(cmds))
	}
}

// Cleanup runs all registered cleanup handlers in reverse order.
func (c *Command) Cleanup() error {
	var err error
	for i := len(c.Cleanups) - 1; i >= 0; i-- {
		err = errors.Join(err, c.Cleanups[i]())
	}

	c.Cleanups = nil
	return err
}
