package cli

import (
	"fmt"
	"slices"
	"strings"
	"text/template"
)

// Renderer formats help and usage text for a command path.
type Renderer func(cmds []*Command) string

// DefaultTemplate renders help output when no command in the current path
// overrides Template.
var DefaultTemplate = `{{define "argDetail"}}{{.Help}} ({{.Type}}{{if .Defaults}}, default: {{range $i, $default := .Defaults}}{{if $i}} | {{end}}{{$default}}{{end}}{{end}}{{if .Choices}}, choices: {{range $i, $choice := .Choices}}{{if $i}}, {{end}}{{$choice}}{{end}}{{end}}{{if .Labels}}, labels: {{range $i, $label := .Labels}}{{if $i}}, {{end}}{{$label}}{{end}}{{end}}){{end}}{{.Usage}}{{if .Command.Help}}

{{.Command.Help}}{{end}}{{range .Sections}}{{if .Args}}

{{if .Current}}Options{{else if .Global}}Options (global){{else}}Options (from {{.Command.Name}}){{end}}:
{{range $i, $arg := .Args}}{{if $i}}{{"\n"}}{{end}}{{if .Option}}{{printf "  --%-14s " .Name}}{{else}}{{printf "  %-16s " .Name}}{{end}}{{template "argDetail" .}}{{end}}{{end}}{{end}}{{if .Command.Commands}}

Subcommands:
{{range $i, $cmd := .Command.Commands}}{{if $i}}{{"\n"}}{{end}}{{printf "  %-16s %s" $cmd.Name $cmd.Help}}{{end}}{{end}}
`

// DefaultRenderer renders help output when no command in the current path
// overrides Renderer.
var DefaultRenderer Renderer = TemplateHelp

// Help formats help and usage text for the provided command path.
func Help(cmds []*Command) string {
	handler := DefaultRenderer
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i].Renderer != nil {
			handler = cmds[i].Renderer
			break
		}
	}

	if handler == nil {
		return ""
	}

	return handler(cmds)
}

// TemplateHelp formats help and usage text with Go's text/template package.
func TemplateHelp(cmds []*Command) string {
	if len(cmds) == 0 {
		return ""
	}

	type helpArg struct {
		Arg      *Arg
		Name     string
		Type     string
		Help     string
		Defaults []string
		Choices  []string
		Labels   []string
		Option   bool
	}

	type helpSection struct {
		Command     *Command
		Current     bool
		Global      bool
		Args        []helpArg
		Positionals []helpArg
		Options     []helpArg
	}

	type helpData struct {
		Commands []*Command
		Command  *Command
		Current  helpSection
		Sections []helpSection
		Usage    string
	}

	sectionFor := func(cmd *Command, current, global bool) helpSection {
		section := helpSection{Command: cmd, Current: current, Global: global}

		for _, arg := range cmd.Args {
			labels := []string{}
			if len(arg.Labels) != 0 {
				keys := make([]string, 0, len(arg.Labels))
				for key := range arg.Labels {
					keys = append(keys, key)
				}
				slices.Sort(keys)

				for _, key := range keys {
					for _, value := range arg.Labels[key] {
						labels = append(labels, key+"="+value)
					}
				}
			}

			item := helpArg{
				Arg:      arg,
				Name:     arg.Name,
				Type:     arg.Type,
				Help:     arg.Help,
				Defaults: arg.Defaults,
				Choices:  arg.Choices,
				Labels:   labels,
			}

			if len(arg.Defaults) == 0 && arg.Type == "bool" {
				item.Defaults = []string{"false"}
			}

			if arg.Positional != 0 {
				section.Positionals = append(section.Positionals, item)
			} else {
				item.Option = true

				if arg.Type == "bool" {
					item.Name = "[no-]" + arg.Name
				}

				section.Options = append(section.Options, item)
			}
		}

		section.Args = append(section.Args, section.Positionals...)
		section.Args = append(section.Args, section.Options...)
		return section
	}

	sections := []helpSection{}
	for i := len(cmds) - 1; i >= 0; i-- {
		sections = append(sections, sectionFor(cmds[i], i == len(cmds)-1, i == 0 && i != len(cmds)-1))
	}

	current := sectionFor(cmds[len(cmds)-1], true, false)
	text := DefaultTemplate
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i].Template != "" {
			text = cmds[i].Template
			break
		}
	}

	usage := &strings.Builder{}
	fmt.Fprint(usage, "Usage:")
	for i, cmd := range cmds {
		fmt.Fprintf(usage, " %s", cmd.Name)
		if i != len(cmds)-1 {
			continue
		}

		for _, arg := range cmd.Args {
			if arg.Positional != 0 {
				fmt.Fprintf(usage, " <%s>", arg.Name)
			}
		}
	}

	w := &strings.Builder{}
	tmpl, err := template.New("help").Parse(text)
	if err != nil {
		panic(fmt.Sprintf("invalid help template: %v", err))
	}

	if err := tmpl.Execute(w, helpData{
		Commands: cmds,
		Command:  cmds[len(cmds)-1],
		Current:  current,
		Sections: sections,
		Usage:    usage.String(),
	}); err != nil {
		panic(fmt.Sprintf("invalid help template: %v", err))
	}

	return w.String()
}
