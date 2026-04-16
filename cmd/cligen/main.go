package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

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
	Resolve    string
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
	Directives []string
	Validate   string
	Required   bool
	Positional int
}

type providerFlags []string

func (p *providerFlags) String() string {
	return strings.Join(*p, ",")
}

func (p *providerFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	*p = append(*p, value)
	return nil
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

			if value, ok := strings.CutPrefix(d, "path="); ok {
				if base := strings.TrimPrefix(arg.Type, "[]"); base != "string" {
					return fmt.Errorf("%s: path directive requires string or []string for %q", c.Path, arg.Name)
				}

				switch value {
				case "exists", "not-exists", "dir", "file", "empty", "mkdir", "creatable", "readable", "writeable", "symlink", "abs", "rel", "exec", "clean", "glob":
				default:
					if !strings.HasPrefix(value, ".") {
						return fmt.Errorf("%s: unsupported path directive %q for %q", c.Path, value, arg.Name)
					}
				}

				if value == "dir" && arg.HasLabel("path", "file") {
					return fmt.Errorf("%s: path directives for %q cannot require both file and dir", c.Path, arg.Name)
				}

				if value == "file" && arg.HasLabel("path", "dir") {
					return fmt.Errorf("%s: path directives for %q cannot require both file and dir", c.Path, arg.Name)
				}

				if value == "file" && arg.HasLabel("path", "empty") {
					return fmt.Errorf("%s: path directives for %q cannot require both file and empty", c.Path, arg.Name)
				}

				if value == "empty" && arg.HasLabel("path", "file") {
					return fmt.Errorf("%s: path directives for %q cannot require both empty and file", c.Path, arg.Name)
				}

				if arg.HasLabel("path", value) {
					return fmt.Errorf("%s: duplicate path directive %q for %q", c.Path, value, arg.Name)
				}

				conflicts := []string{}

				switch value {
				case "dir":
					conflicts = []string{"file", "not-exists", "glob"}
				case "file":
					conflicts = []string{"dir", "empty", "not-exists", "mkdir", "glob"}
				case "empty":
					conflicts = []string{"file", "not-exists", "glob"}
				case "mkdir":
					conflicts = []string{"file", "not-exists", "glob", "symlink"}
				case "not-exists":
					conflicts = []string{"exists", "dir", "file", "empty", "readable", "writeable", "symlink", "exec", "mkdir"}
				case "exists":
					conflicts = []string{"not-exists", "glob"}
				case "readable":
					conflicts = []string{"not-exists", "glob"}
				case "writeable":
					conflicts = []string{"not-exists", "glob"}
				case "symlink":
					conflicts = []string{"not-exists", "mkdir", "glob"}
				case "exec":
					conflicts = []string{"not-exists", "glob"}
				case "creatable":
					conflicts = []string{"glob"}
				case "glob":
					conflicts = []string{"exists", "not-exists", "dir", "file", "empty", "mkdir", "creatable", "readable", "writeable", "symlink", "exec"}
				case "abs":
					conflicts = []string{"rel"}
				case "rel":
					conflicts = []string{"abs"}
				}

				for _, other := range conflicts {
					if arg.HasLabel("path", other) {
						return fmt.Errorf("%s: path directives for %q cannot require both %s and %s", c.Path, arg.Name, value, other)
					}
				}

				if arg.Validate == "" {
					arg.Validate = "cli.PathValidate"
				}

				arg.AddLabel("path", value)
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

func main() {
	gofile := os.Getenv("GOFILE")
	output := ""
	if u := strings.TrimSuffix(gofile, ".go"); u != gofile {
		output = u + ".cli.go"
	}

	src := flag.String("source", gofile, "location of source file with cli.Command definition (defaults to $GOFILE)")
	dst := flag.String("output", output, "output file (defaults to $GOFILE with .cli.go)")
	var providers providerFlags
	flag.Var(&providers, "provider", "native value resolver provider to include in generated code (repeatable)")

	flag.Parse()

	if *src == "" {
		log.Fatal("missing -source parameter or $GOFILE")
	} else {
		gofile = *src
	}

	g, err := Parse(gofile, providers)
	if err != nil {
		log.Fatal(err)
	}

	if *dst == "" {
		output = strings.TrimSuffix(gofile, ".go") + ".cli.go"
	} else {
		output = *dst
	}

	if err := g.Generate(output); err != nil {
		log.Fatal(err)
	}
}
