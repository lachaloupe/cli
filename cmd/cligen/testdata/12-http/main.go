package main

import (
	"net/http"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

const serveTemplate = `{{define "argDetail"}}{{.Help}} ({{.Type}}{{if .Default}}, default: {{.Default}}{{else if .Defaults}}, default: {{range $i, $default := .Defaults}}{{if $i}} | {{end}}{{$default}}{{end}}{{end}}{{if .Choices}}, choices: {{range $i, $choice := .Choices}}{{if $i}}, {{end}}{{$choice}}{{end}}{{end}}{{if .Labels}}, labels: {{range $i, $label := .Labels}}{{if $i}}, {{end}}{{$label}}{{end}}{{end}}){{end}}{{.Usage}}{{if .Command.Help}}

{{.Command.Help}}{{end}}{{if .Current.Options}}

Options:
{{range $i, $arg := .Current.Options}}{{if $i}}{{"\n"}}{{end}}{{printf "  --%-14s " .Name}}{{template "argDetail" .}}{{end}}{{end}}

Routes:
  GET /healthz
  GET /readyz
`

var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name:     "serve",
			Help:     "Run the HTTP server",
			Template: serveTemplate,
			Handler:  cli.MakeServeHTTP(Mux),
		},
	},
}

var Mux = func() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/readyz", func(http.ResponseWriter, *http.Request) {})
	return mux
}()

func main() {
	CLI.Main()
}
