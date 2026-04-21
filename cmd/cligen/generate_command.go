package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

func (gen *Generator) generateCommand(w io.Writer, cmd *Command) {
	if cmd.Name != "" {
		fmt.Fprintf(w, "Name: %q,\n", cmd.Name)
	}

	fmt.Fprintf(w, "Path: %q,\n", cmd.Path)

	if len(cmd.Aliases) != 0 {
		fmt.Fprintf(w, "Aliases: %#v,\n", cmd.Aliases)
	}

	if cmd.Handler != "" {
		fmt.Fprintf(w, "Handler: %s,\n", cmd.Handler)
	}

	if cmd.New != "" {
		fmt.Fprintf(w, "New: %s,\n", cmd.New)
	}

	if cmd.Lookup != "" {
		fmt.Fprintf(w, "Lookup: %s,\n", cmd.Lookup)
	}

	if cmd.Open != "" {
		fmt.Fprintf(w, "Open: %s,\n", cmd.Open)
	}

	help := cmd.Help

	if help == "" {
		help = cmd.Doc
	}

	if help != "" {
		fmt.Fprintf(w, "Help: %q,\n", help)
	}

	if len(cmd.Args) != 0 {
		fmt.Fprintln(w, "Args: []*cli.Arg{")

		for _, arg := range cmd.Args {
			fmt.Fprintln(w, "{")
			fmt.Fprintf(w, "Name:%q,\n", arg.Flag)
			fmt.Fprintf(w, "Type:%q,\n", arg.Type)
			if len(arg.Aliases) != 0 {
				fmt.Fprintf(w, "Aliases:%#v,\n", arg.Aliases)
			}

			if !arg.Native() {
				s := strings.TrimPrefix(arg.Type, "[]")

				fmt.Fprintln(w, "Parse: func(s string) (any, error) {")
				fmt.Fprintf(w, "var v %s\n", s)
				fmt.Fprintln(w, "return v, any(&v).(encoding.TextUnmarshaler).UnmarshalText([]byte(s))")
				fmt.Fprintln(w, "},")
			}

			help := arg.Help

			if help == "" {
				help = arg.Doc
			}

			if help != "" {
				fmt.Fprintf(w, "Help: %q,\n", help)
			}

			switch len(arg.Defaults) {
			case 0:
			case 1:
				fmt.Fprintf(w, "Default: %q,\n", arg.Defaults[0])
			default:
				fmt.Fprintln(w, "Defaults: []string{")
				for _, s := range arg.Defaults {
					fmt.Fprintf(w, "%q,\n", s)
				}
				fmt.Fprintln(w, "},")
			}

			if len(arg.Labels) != 0 {
				keys := make([]string, 0, len(arg.Labels))
				for key := range arg.Labels {
					keys = append(keys, key)
				}

				slices.Sort(keys)

				fmt.Fprintln(w, "Labels: map[string][]string{")
				for _, key := range keys {
					fmt.Fprintf(w, "%q: []string{\n", key)
					for _, value := range arg.Labels[key] {
						fmt.Fprintf(w, "%q,\n", value)
					}
					fmt.Fprintln(w, "},")
				}
				fmt.Fprintln(w, "},")
			}

			if arg.LookupEnum || len(arg.Choices) != 0 {
				choices := arg.Choices
				if arg.LookupEnum {
					fmt.Fprintln(w, "Choices: cli.EnumChoices(func() any {")
					fmt.Fprintf(w, "var v %s\n", strings.TrimPrefix(arg.Type, "[]"))
					fmt.Fprintln(w, "return &v")
					fmt.Fprint(w, "}()")
					for _, choice := range choices {
						fmt.Fprintf(w, ", %q", choice)
					}
					fmt.Fprintln(w, "),")
				} else if len(choices) != 0 {
					fmt.Fprintln(w, "Choices: []string{")
					for _, choice := range choices {
						fmt.Fprintf(w, "%q,\n", choice)
					}
					fmt.Fprintln(w, "},")
				}
			}

			validates := arg.Validates
			if arg.Validate != "" {
				validates = append(validates, arg.Validate)
			}

			if len(validates) != 0 {
				fmt.Fprintf(w, "Validate: func(arg *cli.Arg, s string) error {\n")
				for _, validate := range validates {
					fmt.Fprintf(w, "if err := %s(arg, s); err != nil {\n", validate)
					fmt.Fprintln(w, "return err")
					fmt.Fprintln(w, "}")
				}
				fmt.Fprintln(w, "return nil")
				fmt.Fprintln(w, "},")
			}

			if arg.Required {
				fmt.Fprintln(w, "Required: true,")
			}

			if arg.Positional != 0 {
				fmt.Fprintf(w, "Positional: %d,\n", arg.Positional)
			}

			fmt.Fprintln(w, "},")
		}

		fmt.Fprintln(w, "},")
	}

	if len(cmd.Commands) != 0 {
		fmt.Fprintln(w, "Commands: []*cli.Command{")

		for _, c := range cmd.Commands {
			fmt.Fprintln(w, "{")
			gen.generateCommand(w, c)
			fmt.Fprintln(w, "},")
		}

		fmt.Fprintln(w, "},")
	}
}
