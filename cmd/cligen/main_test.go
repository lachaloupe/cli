package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
			file:       "testdata/04-separated/main.go",
			helpArgs:   []string{"login", "--help"},
			runArgs:    []string{"login", "--user", "alice", "--password", "secret"},
			wantHelp:   "--user",
			wantOutput: "login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			g, err := Parse(tt.file)
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
