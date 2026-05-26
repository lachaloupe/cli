package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"io"
	"slices"
	"strings"
)

func (gen *Generator) generateCommand(w io.Writer, cmd *Command) error {
	if cmd.Name != "" {
		fmt.Fprintf(w, "Name: %q,\n", cmd.Name)
	}

	fmt.Fprintf(w, "Path: %q,\n", cmd.Path)

	if len(cmd.Aliases) != 0 {
		fmt.Fprintf(w, "Aliases: %#v,\n", cmd.Aliases)
	}

	if cmd.LookupExpr != nil {
		expr, err := gen.formatExpr(cmd.LookupExpr)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "Lookup: %s,\n", expr)
	}

	if cmd.OpenExpr != nil {
		expr, err := gen.formatExpr(cmd.OpenExpr)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "Open: %s,\n", expr)
	}

	help := cmd.Help

	if help == "" {
		help = cmd.Doc
	}

	if help != "" {
		fmt.Fprintf(w, "Help: %q,\n", help)
	}

	if cmd.Template != "" {
		fmt.Fprintf(w, "Template: %q,\n", cmd.Template)
	}

	if cmd.RendererExpr != nil {
		expr, err := gen.formatExpr(cmd.RendererExpr)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "Renderer: %s,\n", expr)
	}

	if err := gen.generateCommandInvoke(w, cmd); err != nil {
		return err
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
				if arg.DefaultsOptional[0] {
					fmt.Fprintln(w, "Defaults: []string{")
					fmt.Fprintf(w, "%q,\n", arg.Defaults[0])
					fmt.Fprintln(w, "},")
					fmt.Fprintln(w, "DefaultsOptional: []bool{true},")
				} else {
					fmt.Fprintf(w, "Default: %q,\n", arg.Defaults[0])
				}
			default:
				fmt.Fprintln(w, "Defaults: []string{")
				for _, s := range arg.Defaults {
					fmt.Fprintf(w, "%q,\n", s)
				}
				fmt.Fprintln(w, "},")

				hasOptional := false
				for _, l := range arg.DefaultsOptional {
					if l {
						hasOptional = true
						break
					}
				}

				if hasOptional {
					fmt.Fprintf(w, "DefaultsOptional: []bool{")
					for i, l := range arg.DefaultsOptional {
						if i > 0 {
							fmt.Fprint(w, ", ")
						}
						fmt.Fprintf(w, "%t", l)
					}
					fmt.Fprintln(w, "},")
				}
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
			if err := gen.generateCommand(w, c); err != nil {
				return err
			}
			fmt.Fprintln(w, "},")
		}

		fmt.Fprintln(w, "},")
	}

	return nil
}

func (gen *Generator) generateCommandInvoke(w io.Writer, cmd *Command) error {
	fmt.Fprintln(w, "Invoke: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {")

	if cmd.Struct != "" {
		if cmd.NewExpr != nil {
			expr, err := gen.formatExpr(cmd.NewExpr)
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "s := (%s)()\n", expr)
		} else {
			fmt.Fprintf(w, "s := %s{}\n", cmd.Struct)
		}

		for _, arg := range cmd.Args {
			fmt.Fprintln(w, "")
			fmt.Fprintf(w, "if p := cmd.Get(%q); p != nil && p.Value != nil {\n", arg.Flag)
			fmt.Fprintf(w, "s.%s = p.Value.(%s)\n", arg.Name, arg.Type)
			fmt.Fprintln(w, "}")
		}
	}

	if cmd.HandlerExpr != nil {
		expr, err := gen.formatExpr(cmd.HandlerExpr)
		if err != nil {
			return err
		}

		gen.Imports["errors"] = ""
		fmt.Fprintln(w, "")

		if cmd.Struct == "" {
			fmt.Fprintf(w, "err := errors.Join((%s)(ctx), cmd.Cleanup())\n", expr)
		} else {
			fmt.Fprintf(w, "err := errors.Join((%s)(ctx, s), cmd.Cleanup())\n", expr)
		}

		fmt.Fprintln(w, "if err != nil {")
		fmt.Fprintln(w, "return ctx, err")
		fmt.Fprintln(w, "}")
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "ctx = context.WithValue(ctx, cli.Parent{}, %q)\n", cmd.Path)
	if cmd.Struct != "" {
		fmt.Fprintf(w, "ctx = context.WithValue(ctx, cli.Args(%q), s)\n", cmd.Path)
	}
	fmt.Fprintln(w, "return ctx, nil")
	fmt.Fprintln(w, "},")
	return nil
}

func (gen *Generator) formatExpr(expr ast.Expr) (string, error) {
	w := &strings.Builder{}
	if err := format.Node(w, gen.Fset, expr); err != nil {
		return "", err
	}

	return w.String(), nil
}
