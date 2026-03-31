package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Generator struct {
	Cmds    []*Command
	Imports map[string]struct{}
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
	Default    string
	Directives []string
	Required   bool
	Positional int
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
	case "string":
	case "time.Duration":
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
				arg.Default = value
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

	flag.Parse()

	if *src == "" {
		log.Fatal("missing -source parameter or $GOFILE")
	} else {
		gofile = *src
	}

	g, err := Parse(gofile)
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
