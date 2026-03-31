package cli

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	ErrHelp = errors.New("help")
)

// Args is the type used for adding command arguments to the context.
type Args string

// Parent is the type used for adding the parent's command path to the context.
type Parent struct{}

// Command is used to declare the CLI structure.
type Command struct {
	Name     string
	Aliases  []string
	Path     string
	Help     string
	Handler  any
	New      any
	Args     []*Arg
	Commands []*Command

	invoke func(ctx context.Context, args []string) ([]*Command, error)
}

// Arg is used to declare a CLI argument.
type Arg struct {
	Name       string
	Aliases    []string
	Type       string
	Help       string
	Default    string
	Required   bool
	Positional int
	Parse      func(string) (any, error)
	Value      any
}

func (arg *Arg) SetDefault() error {
	if arg.Default == "" {
		return nil
	}

	if !strings.HasPrefix(arg.Type, "[]") {
		return arg.Set(arg.Default)
	}

	r := csv.NewReader(strings.NewReader(arg.Default))

	lines, err := r.ReadAll()
	if err != nil {
		return err
	}

	if len(lines) != 0 {
		return fmt.Errorf("expected a single CSV line for default value of %q", arg.Name)
	}

	for _, item := range lines[0] {
		if err := arg.Set(item); err != nil {
			return err
		}
	}

	return nil
}

func (arg *Arg) Set(s string) error {
	if item, ok := strings.CutPrefix(arg.Type, "[]"); ok {
		r, err := arg.parse(s, item)
		if err != nil {
			return err
		}

		if arg.Value == nil {
			arg.Value = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(r)), 0, 1).Interface()
		}

		arg.Value = reflect.Append(reflect.ValueOf(arg.Value), reflect.ValueOf(r)).Interface()
		return nil
	}

	val, err := arg.parse(s, arg.Type)
	if err == nil {
		arg.Value = val
	}

	return err
}

func (arg Arg) parse(s, kind string) (any, error) {
	if arg.Parse != nil {
		return arg.Parse(s)
	}

	var r any

	switch kind {
	case "bool":
		if b, err := strconv.ParseBool(s); err != nil {
			return nil, err
		} else {
			r = b
		}
	case "int8":
		if i, err := strconv.ParseInt(s, 10, 8); err != nil {
			return nil, err
		} else {
			r = int8(i)
		}
	case "int16":
		if i, err := strconv.ParseInt(s, 10, 16); err != nil {
			return nil, err
		} else {
			r = int16(i)
		}
	case "int32":
		if i, err := strconv.ParseInt(s, 10, 32); err != nil {
			return nil, err
		} else {
			r = int32(i)
		}
	case "int64":
		if i, err := strconv.ParseInt(s, 10, 64); err != nil {
			return nil, err
		} else {
			r = i
		}
	case "int":
		if i, err := strconv.ParseInt(s, 10, 64); err != nil {
			return nil, err
		} else {
			r = int(i)
		}
	case "uint8":
		if u, err := strconv.ParseUint(s, 10, 8); err != nil {
			return nil, err
		} else {
			r = uint8(u)
		}
	case "uint16":
		if u, err := strconv.ParseUint(s, 10, 16); err != nil {
			return nil, err
		} else {
			r = uint16(u)
		}
	case "uint32":
		if u, err := strconv.ParseUint(s, 10, 32); err != nil {
			return nil, err
		} else {
			r = uint32(u)
		}
	case "uint64":
		if u, err := strconv.ParseUint(s, 10, 64); err != nil {
			return nil, err
		} else {
			r = u
		}
	case "uint":
		if u, err := strconv.ParseUint(s, 10, 64); err != nil {
			return nil, err
		} else {
			r = uint(u)
		}
	case "float32":
		if f, err := strconv.ParseFloat(s, 32); err != nil {
			return nil, err
		} else {
			return float32(f), nil
		}
	case "float64":
		if f, err := strconv.ParseFloat(s, 64); err != nil {
			return nil, err
		} else {
			return float64(f), nil
		}
	case "string":
		r = s
	case "time.Duration":
		if d, err := time.ParseDuration(s); err != nil {
			return nil, err
		} else {
			r = d
		}
	}

	return r, nil
}

func (arg *Arg) Missing() bool {
	if !arg.Required {
		return false
	}

	if arg.Value == nil {
		return true
	}

	if !strings.HasPrefix(arg.Type, "[]") {
		return false
	}

	n := reflect.ValueOf(arg.Value).Len()

	if arg.Positional < 0 {
		return n == 0
	}

	return n != max(1, arg.Positional)
}

func (c *Command) Register(f func(ctx context.Context, args []string) ([]*Command, error)) {
	c.invoke = f
}

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

func (c *Command) Run(args []string) ([]*Command, error) {
	f := c.invoke

	if f == nil {
		panic(fmt.Sprintf("expected cli.Register to be called for %s", c.Name))
	}

	return f(context.Background(), args)
}

func (c *Command) Main() {
	if cmds, err := c.Run(os.Args[1:]); err != nil {
		if err != ErrHelp {
			os.Exit(1)
		}

		fmt.Print(Help(cmds))
	}
}
