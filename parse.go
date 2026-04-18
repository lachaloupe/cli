package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
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

		rest, cmd, err := next.parseArgs(ctx, args)
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

func (c *Command) parseArgs(ctx context.Context, args []string) ([]string, *Command, error) {
	positionals := []*Arg{}

	for _, arg := range c.Args {
		if err := c.applyDefault(ctx, arg); err != nil {
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
			if err := c.setArg(ctx, arg, args[0]); err != nil {
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
				if err := c.setArg(ctx, arg, args[i]); err != nil {
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
					if err := c.setArg(ctx, arg, "true"); err != nil {
						return nil, nil, err
					}
					continue
				}

				if _, err := arg.parse(args[0], arg.Type); err != nil {
					if err := c.setArg(ctx, arg, "true"); err != nil {
						return nil, nil, err
					}
					continue
				}
			}

			if len(args) == 0 {
				return nil, nil, &ParseError{Kind: ErrMissingValue, Name: name}
			}

			if err := c.setArg(ctx, arg, args[0]); err != nil {
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

func (c *Command) applyDefault(ctx context.Context, arg *Arg) error {
	defaults := append([]string{arg.Default}, arg.Defaults...)

	if len(defaults) == 0 {
		return nil
	}

	value := ""
	for _, text := range defaults {
		skip := false
		value = os.Expand(text, func(name string) string {
			if value, ok := os.LookupEnv(name); ok && value != "" {
				return value
			}

			skip = true
			return ""
		})

		if !skip && value != "" {
			break
		}
	}

	if strings.TrimSpace(value) == "" {
		return nil
	}

	if !strings.HasPrefix(arg.Type, "[]") {
		return c.setArgValue(ctx, arg, value, value, false)
	}

	if strings.TrimSpace(value) == "" {
		arg.Value = nil
		return nil
	}

	r := csv.NewReader(strings.NewReader(value))

	lines, err := r.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("invalid CSV default for %q: %v", arg.Name, err))
	}

	if len(lines) != 1 {
		panic(fmt.Sprintf("expected a single CSV line for default value of %q", arg.Name))
	}

	arg.Value = nil

	for _, item := range lines[0] {
		if err := c.setArgValue(ctx, arg, value, item, true); err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) setArg(ctx context.Context, arg *Arg, s string) error {
	raw := s

	if !strings.HasPrefix(arg.Type, "[]") {
		return c.setArgValue(ctx, arg, raw, s, false)
	}

	if !(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		return c.setArgValue(ctx, arg, raw, s, true)
	}

	arg.Value = nil

	line := strings.TrimSpace(s[1 : len(s)-1])
	if line == "" {
		return nil
	}

	r := csv.NewReader(strings.NewReader(line))

	lines, err := r.ReadAll()
	if err != nil {
		return &ArgError{Arg: arg.Name, Value: raw, Err: err}
	}

	if len(lines) != 1 {
		return &ArgError{Arg: arg.Name, Value: raw, Err: fmt.Errorf("expected a single CSV line")}
	}

	for _, item := range lines[0] {
		if err := c.setArgValue(ctx, arg, raw, item, true); err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) setArgValue(ctx context.Context, arg *Arg, raw, s string, appending bool) error {
	kind := arg.Type
	if appending {
		kind = strings.TrimPrefix(kind, "[]")
	}

	if kind == "io.Reader" {
		r, err := c.openReader(ctx, arg, s)
		if err != nil {
			return &ArgError{Arg: arg.Name, Value: raw, Err: err}
		}

		if arg.Validate != nil {
			if err := arg.Validate(arg, s); err != nil {
				return &ArgError{Arg: arg.Name, Value: raw, Err: err}
			}
		}

		if !appending {
			arg.Value = r
			return nil
		}

		if arg.Value == nil {
			arg.Value = []io.Reader{}
		}

		arg.Value = append(arg.Value.([]io.Reader), r)
		return nil
	}

	resolved, err := c.lookupValue(ctx, arg, s)
	if err != nil {
		return &ArgError{Arg: arg.Name, Value: raw, Err: err}
	}
	s = resolved

	if !appending {
		val, err := arg.parse(s, arg.Type)
		if err != nil {
			return &ArgError{Arg: arg.Name, Value: raw, Err: err}
		}

		if arg.Validate != nil {
			if err := arg.Validate(arg, s); err != nil {
				return &ArgError{Arg: arg.Name, Value: raw, Err: err}
			}
		}

		arg.Value = val
		return nil
	}

	r, err := arg.parse(s, kind)
	if err != nil {
		return &ArgError{Arg: arg.Name, Value: raw, Err: err}
	}

	if arg.Validate != nil {
		if err := arg.Validate(arg, s); err != nil {
			return &ArgError{Arg: arg.Name, Value: raw, Err: err}
		}
	}

	if arg.Value == nil {
		arg.Value = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(r)), 0, 1).Interface()
	}

	arg.Value = reflect.Append(reflect.ValueOf(arg.Value), reflect.ValueOf(r)).Interface()
	return nil
}
