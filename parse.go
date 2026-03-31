package cli

import (
	"fmt"
	"slices"
	"strings"
)

func (c *Command) Parse(args []string) ([]*Command, error) {
	list := []*Command{}
	next := c

	for {
		list = append(list, next)

		rest, cmd, err := next.parseNext(args)
		if err != nil {
			return list, err
		}

		for _, arg := range next.Args {
			if arg.Missing() {
				return list, fmt.Errorf("missing required argument %q", arg.Name)
			}
		}

		if cmd == nil {
			return list, nil
		}

		next = cmd
		args = rest
	}
}

func (c *Command) parseNext(args []string) ([]string, *Command, error) {
	positionals := []*Arg{}

	for _, arg := range c.Args {
		if err := arg.SetDefault(); err != nil {
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
			if err := arg.Set(args[0]); err != nil {
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
				if err := arg.Set(args[i]); err != nil {
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
				return nil, nil, fmt.Errorf("unknown flag %q", name)
			}

			if arg.Type == "bool" {
				if len(args) == 0 || strings.HasPrefix(args[0], "-") {
					arg.Set("true")
					continue
				}
			}

			if len(args) == 0 {
				return nil, nil, fmt.Errorf("missing value for flag %q", name)
			}

			if err := arg.Set(args[0]); err != nil {
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

		return nil, nil, fmt.Errorf("unexpected argument %q", args[0])
	}

	for len(args) != 0 {
		if len(positionals) != 0 {
			if err := positional(positionals[0]); err != nil {
				return nil, nil, err
			}

			continue
		}

		return nil, nil, fmt.Errorf("unexpected argument %q", args[0])
	}

	return nil, nil, nil
}
