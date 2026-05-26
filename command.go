package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Args is a context key for the current command's parsed argument struct.
type Args string

// Parent is a context key for the current command's parent path.
type Parent struct{}

// Version enables the built-in version subcommand when set at link time.
var Version string

// Command describes one node in a CLI tree.
type Command struct {
	Name     string
	Aliases  []string
	Path     string
	Help     string
	Template string
	Handler  any
	New      any
	Renderer Renderer
	Invoke   func(context.Context, *Command) (context.Context, error)
	Args     []*Arg
	Commands []*Command
	Lookup   func(context.Context, *Arg, string) (string, bool, error)
	Open     func(context.Context, *Arg, string) (io.Reader, bool, error)
	Cleanups []func() error

	invoke func(ctx context.Context, args []string) ([]*Command, error)
}

// Arg describes one command-line argument.
type Arg struct {
	Name             string
	Aliases          []string
	Type             string
	Help             string
	Default          string
	Defaults         []string
	DefaultsOptional []bool
	Labels           map[string][]string
	Choices          []string
	Required         bool
	Positional       int
	Parse            func(string) (any, error)
	Validate         func(*Arg, string) error
	Value            any
}

// EnumChoices returns the set of allowed enum strings for a value, plus any
// extra literal choices declared alongside it.
func EnumChoices(v any, extras ...string) []string {
	choices := []string{}
	add := func(items []string) {
		for _, item := range items {
			if item == "" || slices.Contains(choices, item) {
				continue
			}

			choices = append(choices, item)
		}
	}

	enum, ok := v.(interface{ Strings() []string })
	if !ok {
		panic(fmt.Sprintf("%T does not implement Strings() []string", v))
	}

	add(enum.Strings())

	add(extras)
	return choices
}

// EnumValidate rejects values that are not listed in the argument's choices.
func EnumValidate(arg *Arg, s string) error {
	if len(arg.Choices) == 0 || slices.Contains(arg.Choices, s) {
		return nil
	}

	return fmt.Errorf("must be one of: %s", strings.Join(arg.Choices, ", "))
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
