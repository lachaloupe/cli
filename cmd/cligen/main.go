package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/lachaloupe/cli"
	"golang.org/x/tools/go/packages"
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
	Cmds      []*Command
	Imports   map[string]struct{}
	Providers map[string]struct{}
}

type Command struct {
	ID         string
	Path       string
	Name       string
	Aliases    []string
	Help       string
	Doc        string
	Directives []string
	Handler    string
	New        string
	Lookup     string
	Open       string
	Struct     string
	Args       []*Arg
	Commands   []*Command
}

type Arg struct {
	Name       string
	Flag       string
	Aliases    []string
	Type       string
	Help       string
	Doc        string
	Defaults   []string
	Labels     map[string][]string
	Choices    []string
	Directives []string
	LookupEnum bool
	Validate   string
	Required   bool
	Positional int
}

func Parse(filename string, providers []string) (*Generator, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName | packages.NeedFiles,
		Dir:  filepath.Dir(filename),
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found in %s", filepath.Dir(filename))
	}

	if len(pkgs[0].Errors) > 0 {
		return nil, pkgs[0].Errors[0]
	}

	gen := &Generator{
		Imports: map[string]struct{}{
			"context":                   {},
			"errors":                    {},
			"github.com/lachaloupe/cli": {},
		},
		Providers: map[string]struct{}{},
	}

	for _, provider := range providers {
		switch provider {
		case "aws":
		default:
			return nil, fmt.Errorf("unsupported provider %q", provider)
		}

		gen.Providers[provider] = struct{}{}
	}

	if _, ok := gen.Providers["aws"]; ok {
		gen.Imports["fmt"] = struct{}{}
		gen.Imports["io"] = struct{}{}
		gen.Imports["net/url"] = struct{}{}
		gen.Imports["strings"] = struct{}{}
		gen.Imports["github.com/aws/aws-sdk-go-v2/aws"] = struct{}{}
		gen.Imports["github.com/aws/aws-sdk-go-v2/config"] = struct{}{}
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/s3"] = struct{}{}
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/secretsmanager"] = struct{}{}
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/ssm"] = struct{}{}
	}

	for i, file := range pkgs[0].GoFiles {
		if f, err := filepath.Abs(file); err != nil || f != filename {
			continue
		}

		file := pkgs[0].Syntax[i]

		for _, decl := range file.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.VAR {
				for _, s := range g.Specs {
					vs, ok := s.(*ast.ValueSpec)
					if !ok {
						continue
					}

					for i, v := range vs.Values {
						lit, ok := v.(*ast.CompositeLit)
						if ok {
							sel, ok := lit.Type.(*ast.SelectorExpr)
							if ok {
								pkg, ok := sel.X.(*ast.Ident)
								if ok && pkg.Name == "cli" && sel.Sel.Name == "Command" {
									cmd := &Command{
										ID:   vs.Names[i].Name,
										Path: "/",
									}

									if err := gen.parseCommand(cmd, lit); err != nil {
										return nil, err
									}

									if err := gen.parseFunctions(pkgs, cmd); err != nil {
										return nil, err
									}

									if err := gen.parseStructs(pkgs, cmd); err != nil {
										return nil, err
									}

									if err := gen.add(cmd); err != nil {
										return nil, err
									}

									break
								}
							}
						}
					}
				}
			}
		}
	}

	if len(gen.Cmds) == 0 {
		return nil, fmt.Errorf("no cli.Command variable found in %s", filename)
	}

	return gen, nil
}

func (arg *Arg) HasLabel(kind, value string) bool {
	return slices.Contains(arg.Labels[kind], value)
}

func (arg *Arg) AddLabel(kind, value string) {
	if arg.Labels == nil {
		arg.Labels = make(map[string][]string)
	}

	arg.Labels[kind] = append(arg.Labels[kind], value)
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
				case c == '_' || c == '-':
					w.WriteByte('-')
				default:
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
					return nil
				}

				arg.Positional = n
				continue
			}

			if value, ok := strings.CutPrefix(d, "default="); ok {
				arg.Defaults = append(arg.Defaults, value)
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

		hasPath := len(arg.Labels["path"]) != 0
		switch {
		case hasPath && (arg.LookupEnum || len(arg.Choices) != 0):
			arg.Validate = `func(arg *cli.Arg, s string) error {
				if err := cli.PathValidate(arg, s); err != nil {
					return err
				}
				return cli.EnumValidate(arg, s)
			}`
		case hasPath:
			arg.Validate = "cli.PathValidate"
		case arg.LookupEnum || len(arg.Choices) != 0:
			arg.Validate = "cli.EnumValidate"
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
			if arg.Positional == -1 {
				return fmt.Errorf("subcommands are not allowed when a positional argument accepts an unbounded number of values")
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
					return fmt.Errorf("%s: duplicate command alias %q for %s and %s", c.Path, name, other, c.Name)
				}

				names[name] = cmd.Name
			}
		}
	}

	return nil
}

func applyPathDirective(cmd *Command, arg *Arg, value string) error {
	conflicts := func(value string) []string {
		switch value {
		case "dir":
			return []string{"file", "not-exists", "glob"}
		case "file":
			return []string{"dir", "empty", "not-exists", "mkdir", "glob"}
		case "empty":
			return []string{"file", "not-exists", "glob"}
		case "mkdir":
			return []string{"file", "not-exists", "glob", "symlink"}
		case "not-exists":
			return []string{"exists", "dir", "file", "empty", "readable", "writeable", "symlink", "exec", "mkdir"}
		case "exists":
			return []string{"not-exists", "glob"}
		case "readable":
			return []string{"not-exists", "glob"}
		case "writeable":
			return []string{"not-exists", "glob"}
		case "symlink":
			return []string{"not-exists", "mkdir", "glob"}
		case "exec":
			return []string{"not-exists", "glob"}
		case "creatable":
			return []string{"glob"}
		case "glob":
			return []string{"exists", "not-exists", "dir", "file", "empty", "mkdir", "creatable", "readable", "writeable", "symlink", "exec"}
		case "abs":
			return []string{"rel"}
		case "rel":
			return []string{"abs"}
		default:
			return nil
		}
	}

	if base := strings.TrimPrefix(arg.Type, "[]"); base != "string" {
		return fmt.Errorf("%s: path directive requires string or []string for %q", cmd.Path, arg.Name)
	}

	switch value {
	case "exists", "not-exists", "dir", "file", "empty", "mkdir", "creatable", "readable", "writeable", "symlink", "abs", "rel", "exec", "clean", "glob":
	default:
		if !strings.HasPrefix(value, ".") {
			return fmt.Errorf("%s: unsupported path directive %q for %q", cmd.Path, value, arg.Name)
		}
	}

	if value == "dir" && arg.HasLabel("path", "file") {
		return fmt.Errorf("%s: path directives for %q cannot require both file and dir", cmd.Path, arg.Name)
	}

	if value == "file" && arg.HasLabel("path", "dir") {
		return fmt.Errorf("%s: path directives for %q cannot require both file and dir", cmd.Path, arg.Name)
	}

	if value == "file" && arg.HasLabel("path", "empty") {
		return fmt.Errorf("%s: path directives for %q cannot require both file and empty", cmd.Path, arg.Name)
	}

	if value == "empty" && arg.HasLabel("path", "file") {
		return fmt.Errorf("%s: path directives for %q cannot require both empty and file", cmd.Path, arg.Name)
	}

	if arg.HasLabel("path", value) {
		return fmt.Errorf("%s: duplicate path directive %q for %q", cmd.Path, value, arg.Name)
	}

	for _, other := range conflicts(value) {
		if arg.HasLabel("path", other) {
			return fmt.Errorf("%s: path directives for %q cannot require both %s and %s", cmd.Path, arg.Name, value, other)
		}
	}

	arg.AddLabel("path", value)
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
		if u := strings.TrimSuffix(gofile, ".go"); u != gofile {
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
