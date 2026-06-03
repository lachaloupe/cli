package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"unicode"

	"github.com/lachaloupe/cli"
)

//go:generate go run . -source main.go -output main.cli.go

var CLI = cli.Command{
	Handler: Run,
}

type Args struct {
	// Location of the source file with the cli.Command definition.
	//cli:default=$GOFILE
	//cli:required
	//cli:path=exists
	//cli:path=file
	//cli:path=.go
	Source string

	// Output file. Defaults to the source file with a .cli.go suffix.
	Output string

	// Native value resolver provider to include in generated code.
	Provider []string
}

type Generator struct {
	Fset          *token.FileSet
	Cmds          []*Command
	Imports       map[string]string
	Package       string
	PackagePath   string
	SourceImports map[string]string
	TypesInfo     *types.Info
	Providers     map[string]struct{}
}

type Command struct {
	ID           string
	Path         string
	Name         string
	Aliases      []string
	Help         string
	Template     string
	Doc          string
	Directives   []string
	RendererExpr ast.Expr
	HandlerExpr  ast.Expr
	HandlerRef   string
	NewExpr      ast.Expr
	LookupExpr   ast.Expr
	OpenExpr     ast.Expr
	Struct       string
	StructName   string
	StructPath   string
	Args         []*Arg
	Commands     []*Command
}

type Arg struct {
	Name             string
	Flag             string
	Aliases          []string
	Type             string
	Help             string
	Doc              string
	Defaults         []string
	DefaultsOptional []bool
	Labels           map[string][]string
	Choices          []string
	Directives       []string
	LookupEnum       bool
	Validate         string
	Validates        []string
	Required         bool
	Positional       int
}

func (arg *Arg) HasLabel(kind, value string) bool {
	for _, label := range arg.Labels[kind] {
		if label == value {
			return true
		}

		if name, _, ok := strings.Cut(label, ":"); ok && name == value {
			return true
		}
	}
	return false
}

func (arg *Arg) Native() bool {
	switch strings.TrimPrefix(arg.Type, "[]") {
	case "bool":
	case "int8":
	case "int16":
	case "int32":
	case "int64":
	case "int":
	case "uint8":
	case "uint16":
	case "uint32":
	case "uint64":
	case "uint":
	case "float32":
	case "float64":
	case "complex64":
	case "complex128":
	case "string":
	case "time.Duration":
	case "net.HardwareAddr":
	case "net.IPNet":
	case "url.URL":
	case "mail.Address":
	case "io.Reader":
	default:
		return false
	}

	return true
}

func (c *Command) CommandList() []*Command {
	p := []*Command{c}

	for _, cmd := range c.Commands {
		p = append(p, cmd.CommandList()...)
	}

	return p
}

func (c *Command) Process() error {
	for _, d := range c.Directives {
		if value, ok := strings.CutPrefix(d, "alias="); ok {
			if value == "" {
				return fmt.Errorf("%s: alias cannot be empty", c.Path)
			}

			c.Aliases = append(c.Aliases, value)
		}
	}

	for _, arg := range c.Args {
		if arg.Flag == "" {
			arg.Flag = arg.Name
		}

		if arg.Positional == 0 {
			w := strings.Builder{}

			s := []rune(arg.Name)
			for i, c := range s {
				switch {
				case unicode.IsUpper(c):
					if i > 0 {
						last := s[i-1]
						down := i+1 < len(s) && unicode.IsLower(s[i+1])
						if unicode.IsLower(last) || unicode.IsDigit(last) || (unicode.IsUpper(last) && down) {
							w.WriteByte('-')
						}
					}

					w.WriteRune(unicode.ToLower(c))
				case unicode.IsDigit(c):
					if i > 0 && unicode.IsLetter(s[i-1]) && !unicode.IsUpper(s[i-1]) {
						w.WriteByte('-')
					}

					w.WriteRune(c)
				case c == '_' || c == '-':
					w.WriteByte('-')
				default:
					if i > 0 && unicode.IsDigit(s[i-1]) {
						w.WriteByte('-')
					}

					w.WriteRune(unicode.ToLower(c))
				}
			}

			arg.Flag = w.String()
		}

		for _, d := range arg.Directives {
			if d == "arg" {
				if strings.HasPrefix(arg.Type, "[]") {
					arg.Positional = -1
				} else {
					arg.Positional = 1
				}

				continue
			}

			if d == "required" {
				arg.Required = true
				continue
			}

			if count, ok := strings.CutPrefix(d, "arg="); ok {
				n, err := strconv.Atoi(count)
				if err != nil {
					return fmt.Errorf("%s: invalid arg count %q for %q", c.Path, count, arg.Name)
				}

				arg.Positional = n
				continue
			}

			if value, ok := strings.CutPrefix(d, "default?="); ok {
				arg.Defaults = append(arg.Defaults, value)
				arg.DefaultsOptional = append(arg.DefaultsOptional, true)
				continue
			}

			if value, ok := strings.CutPrefix(d, "default="); ok {
				arg.Defaults = append(arg.Defaults, value)
				arg.DefaultsOptional = append(arg.DefaultsOptional, false)
				continue
			}

			if d == "enum" {
				arg.LookupEnum = true
				continue
			}

			if value, ok := strings.CutPrefix(d, "enum="); ok {
				if value == "" {
					return fmt.Errorf("%s: enum value for %q cannot be empty", c.Path, arg.Name)
				}

				arg.Choices = append(arg.Choices, value)
				continue
			}

			if value, ok := strings.CutPrefix(d, "path="); ok {
				if err := applyPathDirective(c, arg, value); err != nil {
					return err
				}
				continue
			}

			if value, ok := strings.CutPrefix(d, "alias="); ok {
				if value == "" {
					return fmt.Errorf("%s: alias for %q cannot be empty", c.Path, arg.Name)
				}

				arg.Aliases = append(arg.Aliases, value)
				continue
			}
		}

		if len(arg.Labels["path"]) != 0 {
			arg.Validates = append(arg.Validates, "cli.PathValidate")
		}

		if arg.LookupEnum || len(arg.Choices) != 0 {
			arg.Validates = append(arg.Validates, "cli.EnumValidate")
		}
	}

	names := map[string]string{
		"h":    "help",
		"help": "help",
	}

	for _, arg := range c.Args {
		if arg.Positional != 0 {
			continue
		}

		for _, name := range append([]string{arg.Flag}, arg.Aliases...) {
			if name == "" {
				continue
			}

			if other, ok := names[name]; ok {
				return fmt.Errorf("%s: duplicate flag alias %q for %s and %s", c.Path, name, other, arg.Flag)
			}

			names[name] = arg.Flag
		}
	}

	if len(c.Commands) != 0 {
		for _, arg := range c.Args {
			if arg.Positional != 0 && !arg.Required {
				return fmt.Errorf("positional argument %q must be required when subcommands are present", arg.Flag)
			}
		}

		for _, cmd := range c.Commands {
			if err := cmd.Process(); err != nil {
				return err
			}
		}

		names := make(map[string]string)

		for _, cmd := range c.Commands {
			for _, name := range append([]string{cmd.Name}, cmd.Aliases...) {
				if name == "" {
					continue
				}

				if other, ok := names[name]; ok {
					return fmt.Errorf("%s: duplicate command alias %q for %s and %s", c.Path, name, other, cmd.Name)
				}

				names[name] = cmd.Name
			}
		}
	}

	return nil
}

func applyPathDirective(cmd *Command, arg *Arg, value string) error {
	conflicts := func(value string) []string {
		value, _, _ = strings.Cut(value, ":")

		switch value {
		case "dir":
			return []string{"dir", "file", "not-exists", "glob"}
		case "file":
			return []string{"file", "dir", "empty", "not-exists", "glob"}
		case "empty":
			return []string{"empty", "file", "not-exists", "glob"}
		case "mkdir":
			return []string{"mkdir", "not-exists", "glob", "symlink"}
		case "not-exists":
			return []string{"not-exists", "exists", "dir", "file", "empty", "symlink", "exec", "mkdir"}
		case "exists":
			return []string{"exists", "not-exists", "glob"}
		case "symlink":
			return []string{"symlink", "not-exists", "mkdir", "glob"}
		case "exec":
			return []string{"exec", "not-exists", "glob"}
		case "glob":
			return []string{"glob", "exists", "not-exists", "dir", "file", "empty", "mkdir", "symlink", "exec"}
		case "abs":
			return []string{"abs", "rel"}
		case "rel":
			return []string{"rel", "abs"}
		default:
			return []string{value}
		}
	}

	if base := strings.TrimPrefix(arg.Type, "[]"); base != "string" {
		return fmt.Errorf("%s: path directive requires string or []string for %q", cmd.Path, arg.Name)
	}

	switch value {
	case "exists", "not-exists", "dir", "file", "empty", "mkdir", "symlink", "abs", "rel", "exec", "clean", "glob":
	default:
		if after, ok := strings.CutPrefix(value, "mkdir:"); ok {
			if _, err := strconv.ParseUint(after, 8, 32); err != nil {
				return fmt.Errorf("%s: invalid mkdir mode %q for %q", cmd.Path, after, arg.Name)
			}

			break
		}

		if !strings.HasPrefix(value, ".") {
			return fmt.Errorf("%s: unsupported path directive %q for %q", cmd.Path, value, arg.Name)
		}
	}

	for i, other := range conflicts(value) {
		if !arg.HasLabel("path", other) {
			continue
		}

		if i == 0 {
			return fmt.Errorf("%s: duplicate path directive %q for %q", cmd.Path, value, arg.Name)
		}

		return fmt.Errorf("%s: path directives for %q cannot require both %s and %s", cmd.Path, arg.Name, value, other)
	}

	if arg.Labels == nil {
		arg.Labels = make(map[string][]string)
	}

	arg.Labels["path"] = append(arg.Labels["path"], value)
	return nil
}

// Run generates the CLI glue for the requested source file.
func Run(ctx context.Context, args Args) error {
	_ = ctx

	gofile := args.Source
	g, err := Parse(gofile, args.Provider)
	if err != nil {
		return err
	}

	output := args.Output
	if output == "" {
		if u, ok := strings.CutSuffix(gofile, ".go"); ok {
			output = u + ".cli.go"
		}
	}

	if err := g.Generate(output); err != nil {
		return err
	}

	return nil
}

func main() {
	CLI.Main()
}
