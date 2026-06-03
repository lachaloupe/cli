package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamples(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOTELEMETRY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")

	tests := []struct {
		name      string
		providers []string
	}{
		{name: "01-minimal"},
		{name: "02-simple"},
		{name: "03-commands"},
		{name: "04-docker"},
		{name: "05-curl"},
		{name: "06-git"},
		{name: "07-cp"},
		{name: "08-head"},
		{name: "09-percentile"},
		{name: "10-sync"},
		{name: "11-aws", providers: []string{"aws"}},
		{name: "12-http"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gen, err := Parse(filepath.Join("testdata", test.name, "main.go"), test.providers)
			if err != nil {
				t.Fatal(err)
			}

			output := filepath.Join(t.TempDir(), "main.cli.go")
			if err := gen.Generate(output); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFailures(t *testing.T) {
	tests := []struct {
		name      string
		providers []string
		want      string
	}{
		{
			name: "bad-handler-signature",
			want: `/: expecting Handler to be func(context.Context[, arg SomeArgs]) error`,
		},
		{
			name: "bad-help",
			want: `/: expected Help: "text"`,
		},
		{
			name: "bad-name",
			want: `/: invalid Name: "$"`,
		},
		{
			name: "bad-template",
			want: `/: expected Template: "text"`,
		},
		{
			name: "duplicate-command-alias",
			want: `/: duplicate command alias "push" for publish and push`,
		},
		{
			name: "duplicate-flag-alias",
			want: `/: duplicate flag alias "x" for one and two`,
		},
		{
			name: "duplicate-help-alias",
			want: `/: duplicate flag alias "help" for help and verbose`,
		},
		{
			name: "empty-commands",
			want: `/: expecting Commands: []*cli.Command{}`,
		},
		{
			name: "empty-command-alias",
			want: `/: alias cannot be empty`,
		},
		{
			name: "empty-flag-alias",
			want: `/: alias for "Path" cannot be empty`,
		},
		{
			name: "empty-enum",
			want: `/: enum value for "Format" cannot be empty`,
		},
		{
			name: "embedded-field",
			want: `Args: unsupported field: embedded fields are not supported`,
		},
		{
			name: "handler-missing-error",
			want: `/: expecting Handler to be func(context.Context[, arg SomeArgs]) error`,
		},
		{
			name: "handler-pointer-args",
			want: `/: expecting Handler to be func(context.Context[, arg SomeArgs]) error`,
		},
		{
			name: "invalid-arg-count",
			want: `/: invalid arg count "nope" for "Path"`,
		},
		{
			name: "missing-command",
			want: `no cli.Command variable found in main.go`,
		},
		{
			name: "multiple-field-names",
			want: `Args: unsupported field: multiple field names are not supported`,
		},
		{
			name: "path-conflict",
			want: `/: path directives for "Path" cannot require both dir and file`,
		},
		{
			name: "path-directive-on-int",
			want: `/: path directive requires string or []string for "Count"`,
		},
		{
			name: "subcommands-with-varargs",
			want: `positional argument "paths" must be required when subcommands are present`,
		},
		{
			name: "unsupported-field",
			want: `Args: unsupported field: "Labels"`,
		},
		{
			name: "unsupported-path-directive",
			want: `/: unsupported path directive "outside" for "Path"`,
		},
		{
			name:      "unsupported-provider",
			providers: []string{"gcp"},
			want:      `unsupported provider "gcp"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			root, err := filepath.Abs("../..")
			if err != nil {
				t.Fatal(err)
			}

			source, err := os.ReadFile(filepath.Join("testdata", "failures", test.name, "main.go"))
			if err != nil {
				t.Fatal(err)
			}

			mod := "module example.com/cligenfail\n\n" +
				"go 1.26.3\n\n" +
				"require github.com/lachaloupe/cli v0.0.0\n\n" +
				"replace github.com/lachaloupe/cli => " + filepath.ToSlash(root) + "\n"

			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(dir, "main.go")
			if err := os.WriteFile(filename, source, 0644); err != nil {
				t.Fatal(err)
			}

			_, err = Parse(filename, test.providers)
			if err == nil {
				t.Fatal("expected error")
			}

			got := strings.ReplaceAll(err.Error(), filename, "main.go")
			if got != test.want {
				t.Fatalf("got error:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
}
