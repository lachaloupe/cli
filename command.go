package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Args is a context key used to store the current command arguments.
type Args string

// Parent is a context key used to store the parent command path.
type Parent struct{}

// Command describes a command in a CLI tree, including its flags,
// positional arguments, subcommands, and runtime handler hooks.
type Command struct {
	// Name is the command name used on the command line.
	Name string
	// Aliases lists additional names that can invoke the command.
	Aliases []string
	// Path is the full command path used when nesting commands.
	Path string
	// Help is the short help text shown in generated usage output.
	Help string
	// Handler stores the user-defined handler value associated with the command.
	Handler any
	// New stores the constructor used to allocate handler input values.
	New any
	// Args lists the flags and positional arguments accepted by the command.
	Args []*Arg
	// Commands lists the command's direct subcommands.
	Commands []*Command
	// Resolve rewrites raw argument values before built-in file resolution and parsing.
	Resolve func(context.Context, *Arg, string) (string, bool, error)

	invoke func(ctx context.Context, args []string) ([]*Command, error)
}

// Arg describes a single command-line argument, whether it is exposed as
// a flag or consumed positionally.
type Arg struct {
	// Name is the flag or positional argument name.
	Name string
	// Aliases lists additional flag names accepted for the argument.
	Aliases []string
	// Type is the Go type name used for parsing the argument value.
	Type string
	// Help is the help text shown for the argument in usage output.
	Help string
	// Default is the primary default value template for the argument.
	Default string
	// Defaults lists fallback default value templates evaluated in order.
	Defaults []string
	// Labels stores validation and metadata labels grouped by category.
	Labels map[string][]string
	// Required reports whether the argument must be provided.
	Required bool
	// Positional controls whether the argument is positional and how many values it consumes.
	Positional int
	// Parse overrides the built-in parser for converting raw strings into values.
	Parse func(string) (any, error)
	// Validate validates each raw argument value after parsing.
	Validate func(*Arg, string) error
	// Value stores the parsed argument value.
	Value any
}

// SetDefault applies the first non-empty default value configured for the argument.
func (arg *Arg) SetDefault(ctx context.Context, resolve func(context.Context, *Arg, string) (string, error)) error {
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

	if value == "" {
		return nil
	}

	if !strings.HasPrefix(arg.Type, "[]") {
		return arg.Set(ctx, resolve, value)
	}

	r := csv.NewReader(strings.NewReader(value))

	lines, err := r.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("invalid CSV default for %q: %v", arg.Name, err))
	}

	if len(lines) != 1 {
		panic(fmt.Sprintf("expected a single CSV line for default value of %q", arg.Name))
	}

	for _, item := range lines[0] {
		if err := arg.Set(ctx, resolve, item); err != nil {
			return err
		}
	}

	return nil
}

// Set parses, validates, and stores a raw argument value.
func (arg *Arg) Set(ctx context.Context, resolve func(context.Context, *Arg, string) (string, error), s string) error {
	raw := s
	resolved, err := resolve(ctx, arg, s)
	if err != nil {
		return &ArgError{Arg: arg.Name, Value: raw, Err: err}
	}
	s = resolved

	if item, ok := strings.CutPrefix(arg.Type, "[]"); ok {
		r, err := arg.parse(s, item)
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
	case "complex64":
		if c, err := strconv.ParseComplex(s, 64); err != nil {
			return nil, err
		} else {
			return complex64(c), nil
		}
	case "complex128":
		if c, err := strconv.ParseComplex(s, 128); err != nil {
			return nil, err
		} else {
			return c, nil
		}
	case "string":
		r = s
	case "time.Duration":
		if d, err := time.ParseDuration(s); err != nil {
			return nil, err
		} else {
			r = d
		}
	case "net.HardwareAddr":
		if hw, err := net.ParseMAC(s); err != nil {
			return nil, err
		} else {
			r = hw
		}
	case "net.IPNet":
		if _, ipnet, err := net.ParseCIDR(s); err != nil {
			return nil, err
		} else {
			r = *ipnet
		}
	case "url.URL":
		if u, err := url.Parse(s); err != nil {
			return nil, err
		} else {
			r = *u
		}
	case "mail.Address":
		if addr, err := mail.ParseAddress(s); err != nil {
			return nil, err
		} else {
			r = *addr
		}
	default:
		panic(fmt.Sprintf("unsupported argument type %q", kind))
	}

	return r, nil
}

// Missing reports whether a required argument is still unset.
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
	if cmds, err := c.Run(context.Background(), os.Args[1:]); err != nil {
		if err != ErrHelp {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, Help(cmds))
	}
}

func (c *Command) resolveValue(ctx context.Context, arg *Arg, s string) (string, error) {
	if strings.HasPrefix(s, "@@") {
		return s[1:], nil
	}

	if c.Resolve != nil {
		if value, ok, err := c.Resolve(ctx, arg, s); err != nil {
			return "", err
		} else if ok {
			return value, nil
		}
	}

	if !strings.HasPrefix(s, "@") {
		return s, nil
	}

	p := strings.TrimPrefix(s, "@")
	if p == "" {
		return "", fmt.Errorf("%s: empty file reference", arg.Name)
	}

	body, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("%s: read %q: %w", arg.Name, p, err)
	}

	return string(body), nil
}
