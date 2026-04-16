package cli

import (
	"context"
	"slices"
	"strings"
)

// Parse walks the command tree, parses arguments, and returns the matched command path.
func (c *Command) Parse(ctx context.Context, args []string) ([]*Command, error) {
	list := []*Command{}
	next := c

	for _, cmd := range c.CommandList() {
		cmd.Cleanups = nil
	}

	for {
		list = append(list, next)

		rest, cmd, err := next.parseNext(ctx, args)
		if err != nil {
			return list, err
		}

		for _, arg := range next.Args {
			if arg.Missing() {
				return list, &ParseError{Kind: ErrMissingRequired, Name: arg.Name}
			}
		}

		if cmd == nil {
			return list, nil
		}

		next = cmd
		args = rest
	}
}

func (c *Command) parseNext(ctx context.Context, args []string) ([]string, *Command, error) {
	positionals := []*Arg{}

	for _, arg := range c.Args {
		if err := arg.SetDefault(ctx, c.resolveValue, c.resolveReader); err != nil {
			return nil, nil, err
		}

		if arg.Positional != 0 {
			positionals = append(positionals, arg)
		}
	}

	positional := func(arg *Arg) error {
		positionals = positionals[1:]

		n := arg.Positional

		if n == 1 {
			if err := arg.Set(ctx, c.resolveValue, c.resolveReader, args[0]); err != nil {
				return err
			}
		} else {
			if n < 0 {
				left := 0
				for _, arg := range positionals {
					if arg.Positional < 0 {
						panic("cannot have multiple unbounded positionals")
					}

					if arg.Positional > 0 {
						left += arg.Positional
					}
				}

				n = max(0, len(args)-left)
			} else {
				n = min(n, len(args))
			}

			for i := range n {
				if err := arg.Set(ctx, c.resolveValue, c.resolveReader, args[i]); err != nil {
					return err
				}
			}
		}

		args = args[n:]
		return nil
	}

	for len(args) != 0 {
		if args[0] == "-h" || args[0] == "--help" {
			return nil, nil, ErrHelp
		}

		if args[0] == "--" {
			args = args[1:]
			break
		}

		if name, ok := strings.CutPrefix(args[0], "-"); ok {
			args = args[1:]
			name = strings.TrimPrefix(name, "-")

			if lhs, rhs, ok := strings.Cut(name, "="); ok {
				name = lhs
				args = append([]string{rhs}, args...)
			}

			arg := c.Get(name)
			if arg == nil {
				return nil, nil, &ParseError{Kind: ErrUnknownFlag, Name: name}
			}

			if arg.Type == "bool" {
				if len(args) == 0 || strings.HasPrefix(args[0], "-") {
					if err := arg.Set(ctx, c.resolveValue, c.resolveReader, "true"); err != nil {
						return nil, nil, err
					}
					continue
				}
			}

			if len(args) == 0 {
				return nil, nil, &ParseError{Kind: ErrMissingValue, Name: name}
			}

			if err := arg.Set(ctx, c.resolveValue, c.resolveReader, args[0]); err != nil {
				return nil, nil, err
			}

			args = args[1:]
			continue
		}

		if len(positionals) != 0 {
			if err := positional(positionals[0]); err != nil {
				return nil, nil, err
			}

			continue
		}

		for _, cmd := range c.Commands {
			if cmd.Name == args[0] {
				return args[1:], cmd, nil
			}

			if slices.Contains(cmd.Aliases, args[0]) {
				return args[1:], cmd, nil
			}
		}

		return nil, nil, &ParseError{Kind: ErrUnexpectedArg, Value: args[0]}
	}

	for len(args) != 0 {
		if len(positionals) != 0 {
			if err := positional(positionals[0]); err != nil {
				return nil, nil, err
			}

			continue
		}

		return nil, nil, &ParseError{Kind: ErrUnexpectedArg, Value: args[0]}
	}

	return nil, nil, nil
}
