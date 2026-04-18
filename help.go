package cli

import (
	"fmt"
	"slices"
	"strings"
)

// Help formats help and usage text for the provided command path.
func Help(cmds []*Command) string {
	w := &strings.Builder{}

	fmt.Fprint(w, "Usage:")

	for i, c := range cmds {
		if len(c.Args) == 0 {
			fmt.Fprintf(w, " %s", c.Name)
		} else {
			fmt.Fprintf(w, " %s", c.Name)

			if i == len(cmds)-1 {
				for _, arg := range c.Args {
					if arg.Positional != 0 {
						fmt.Fprintf(w, " <%s>", arg.Name)
					}
				}
			}
		}
	}

	fmt.Fprintln(w)

	last := cmds[len(cmds)-1]
	if last.Help != "" {
		fmt.Fprintf(w, "\n%s\n", last.Help)
	}

	for i := range len(cmds) {
		n := len(cmds) - 1 - i
		c := cmds[n]

		if len(c.Args) == 0 {
			continue
		}

		fmt.Fprintln(w)

		switch {
		case n == len(cmds)-1:
			fmt.Fprintf(w, "Options:\n")
		case n == 0:
			fmt.Fprintf(w, "Options (global):\n")
		default:
			fmt.Fprintf(w, "Options (from %s):\n", c.Name)
		}

		opts, args := []*Arg{}, []*Arg{}

		for _, arg := range c.Args {
			if arg.Positional == 0 {
				opts = append(opts, arg)
			} else {
				args = append(args, arg)
			}
		}

		for _, arg := range args {
			fmt.Fprintf(w, "  %-16s %s\n", arg.Name, arg.detail())
		}

		for _, arg := range opts {
			fmt.Fprintf(w, "  --%-14s %s\n", arg.Name, arg.detail())
		}
	}

	if len(last.Commands) != 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Subcommands:")

		for _, c := range last.Commands {
			fmt.Fprintf(w, "  %-16s %s\n", c.Name, c.Help)
		}
	}

	return w.String()
}

func (arg *Arg) detail() string {
	detail := fmt.Sprintf("%s (%s", arg.Help, arg.Type)

	if arg.Default != "" && len(arg.Defaults) != 0 {
		panic("arg cannot have both Default and Defaults")
	}

	switch {
	case arg.Default != "":
		detail += fmt.Sprintf(", default: %s", arg.Default)
	case len(arg.Defaults) != 0:
		detail += fmt.Sprintf(", default: %s", strings.Join(arg.Defaults, " | "))
	}

	if len(arg.Choices) != 0 {
		detail += fmt.Sprintf(", choices: %s", strings.Join(arg.Choices, ", "))
	}

	if len(arg.Labels) != 0 {
		keys := make([]string, 0, len(arg.Labels))
		for key := range arg.Labels {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		items := []string{}
		for _, key := range keys {
			for _, value := range arg.Labels[key] {
				items = append(items, key+"="+value)
			}
		}

		detail += fmt.Sprintf(", labels: %s", strings.Join(items, ", "))
	}

	return detail + ")"
}
