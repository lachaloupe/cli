package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCLIs(t *testing.T) {
	t.Setenv("GOWORK", "off")

	tests := []struct {
		file       string
		helpArgs   []string
		runArgs    []string
		wantHelp   string
		wantOutput string
	}{
		{
			file:       "testdata/01-minimal/main.go",
			helpArgs:   []string{"--help"},
			runArgs:    []string{"--some-flag", "hello"},
			wantHelp:   "Usage:",
			wantOutput: "{hello}",
		},
		{
			file:       "testdata/02-simple/main.go",
			helpArgs:   []string{"--help"},
			runArgs:    []string{"-i", "-r", "-m", "2", "demo"},
			wantHelp:   "default: 10",
			wantOutput: "{demo [] true false true 2}",
		},
		{
			file:       "testdata/03-commands/main.go",
			helpArgs:   []string{"signout", "--help"},
			runArgs:    []string{"signout"},
			wantHelp:   "logout",
			wantOutput: "logout",
		},
		{
			file:     "testdata/04-docker/main.go",
			helpArgs: []string{"container", "run", "--help"},
			runArgs: []string{
				"container", "run",
				"--detach",
				"--env", "APP_ENV=dev",
				"--publish", "8080:80",
				"--name", "web",
				"nginx:latest",
				"echo", "hello",
			},
			wantHelp:   "--publish",
			wantOutput: "{true [APP_ENV=dev] [] false web [8080:80] false missing false []  nginx:latest [echo hello]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			g, err := Parse(tt.file, nil)
			if err != nil {
				t.Fatal(err)
			}

			generated := strings.ReplaceAll(tt.file, ".go", ".cli.go")
			if err := g.Generate(generated); err != nil {
				t.Fatal(err)
			}

			dir := filepath.Dir(tt.file)
			binary := filepath.Join(t.TempDir(), filepath.Base(dir))
			cache := filepath.Join(t.TempDir(), "gocache")

			build := exec.Command("go", "build", "-o", binary, ".")
			build.Dir = dir
			build.Env = append(os.Environ(), "GOWORK=off", "GOCACHE="+cache)
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("go build failed: %v\n%s", err, out)
			}

			help := exec.Command(binary, tt.helpArgs...)
			help.Dir = dir
			if out, err := help.CombinedOutput(); err != nil {
				t.Fatalf("help failed: %v\n%s", err, out)
			} else if !strings.Contains(string(out), tt.wantHelp) {
				t.Fatalf("help output missing %q:\n%s", tt.wantHelp, out)
			}

			run := exec.Command(binary, tt.runArgs...)
			run.Dir = dir
			if out, err := run.CombinedOutput(); err != nil {
				t.Fatalf("run failed: %v\n%s", err, out)
			} else if !strings.Contains(string(out), tt.wantOutput) {
				t.Fatalf("run output missing %q:\n%s", tt.wantOutput, out)
			}
		})
	}
}

func TestParseProviderValidation(t *testing.T) {
	if _, err := Parse("testdata/01-minimal/main.go", []string{"nope"}); err == nil {
		t.Fatal("expected unsupported provider to fail")
	}
}

func TestGenerateProviderAWS(t *testing.T) {
	g := &Generator{
		Imports: map[string]struct{}{
			"context":                             {},
			"errors":                              {},
			"fmt":                                 {},
			"io":                                  {},
			"net/url":                             {},
			"strings":                             {},
			"github.com/lachaloupe/cli":           {},
			"github.com/aws/aws-sdk-go-v2/aws":    {},
			"github.com/aws/aws-sdk-go-v2/config": {},
			"github.com/aws/aws-sdk-go-v2/service/s3":             {},
			"github.com/aws/aws-sdk-go-v2/service/ssm":            {},
			"github.com/aws/aws-sdk-go-v2/service/secretsmanager": {},
		},
		Providers: map[string]struct{}{
			"aws": {},
		},
		Cmds: []*Command{
			{
				ID:      "CLI",
				Path:    "/",
				Handler: "Run",
				Args: []*Arg{
					{
						Name: "name",
						Flag: "name",
						Type: "string",
					},
				},
			},
		},
	}

	output := "provider_test_output.go"
	t.Cleanup(func() {
		_ = os.Remove(output)
	})

	if err := g.Generate(output); err != nil {
		t.Fatal(err)
	}

	p, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}

	text := string(p)
	for _, want := range []string{
		`func resolveNativeValue(ctx context.Context, arg *cli.Arg, value string) (string, bool, error) {`,
		`if root.Resolve != nil {`,
		`prevResolve := root.Resolve`,
		`root.Resolve = resolveNativeValue`,
		`return resolveNativeValue(ctx, arg, value)`,
		`if root.ResolveReader != nil {`,
		`prevResolveReader := root.ResolveReader`,
		`root.ResolveReader = resolveNativeReader`,
		`func resolveNativeReader(ctx context.Context, arg *cli.Arg, value string) (io.Reader, bool, error) {`,
		`errors.Join(f(ctx), cmd.Cleanup())`,
		`"github.com/aws/aws-sdk-go-v2/config"`,
		`"github.com/aws/aws-sdk-go-v2/service/s3"`,
		`"io"`,
		`"net/url"`,
		`strings.HasPrefix(value, "@aws:")`,
		`if u.Scheme != "s3" {`,
		`case "ssm":`,
		`case "secret":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated output missing %q:\n%s", want, text)
		}
	}
}

func TestArgNativeStandardTypes(t *testing.T) {
	for _, typ := range []string{
		"complex64",
		"complex128",
		"net.HardwareAddr",
		"net.IPNet",
		"url.URL",
		"mail.Address",
		"io.Reader",
		"[]complex128",
		"[]net.HardwareAddr",
		"[]io.Reader",
	} {
		arg := &Arg{Type: typ}
		if !arg.Native() {
			t.Fatalf("%s should be native", typ)
		}
	}
}

func TestProcessPathDirectiveSetsDefaultValidate(t *testing.T) {
	c := &Command{
		Path: "/",
		Args: []*Arg{
			{
				Name:       "Input",
				Type:       "string",
				Directives: []string{"path=creatable", "path=.txt", "path=clean"},
			},
		},
	}

	if err := c.Process(); err != nil {
		t.Fatal(err)
	}

	arg := c.Args[0]
	if want, got := "cli.PathValidate", arg.Validate; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}

	if want, got := []string{"creatable", ".txt", "clean"}, arg.Labels["path"]; !slices.Equal(got, want) {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestProcessPathDirectivePreservesValidate(t *testing.T) {
	c := &Command{
		Path: "/",
		Args: []*Arg{
			{
				Name:       "Input",
				Type:       "string",
				Validate:   "customValidate",
				Directives: []string{"path=exists"},
			},
		},
	}

	if err := c.Process(); err != nil {
		t.Fatal(err)
	}

	if want, got := "customValidate", c.Args[0].Validate; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestProcessPathDirectiveRejectsConflicts(t *testing.T) {
	tests := []struct {
		name       string
		directives []string
	}{
		{name: "not-exists with file", directives: []string{"path=not-exists", "path=file"}},
		{name: "mkdir with file", directives: []string{"path=mkdir", "path=file"}},
		{name: "glob with exists", directives: []string{"path=glob", "path=exists"}},
		{name: "abs with rel", directives: []string{"path=abs", "path=rel"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				Path: "/",
				Args: []*Arg{
					{
						Name:       "Input",
						Type:       "string",
						Directives: tt.directives,
					},
				},
			}

			if err := c.Process(); err == nil {
				t.Fatal("expected conflicting path directives to fail")
			}
		})
	}
}
