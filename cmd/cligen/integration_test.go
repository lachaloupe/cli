package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUsingExamples(t *testing.T) {
	dir := t.TempDir()
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"
	relFrom := func(base, target string) string {
		rel, err := filepath.Rel(base, target)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS == "windows" {
			rel = filepath.ToSlash(rel)
		}
		return rel
	}

	cov, err := filepath.Abs("./.coverdata")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(cov, 0755); err != nil {
		t.Fatal(err)
	}

	copyManifest := filepath.Join(dir, "manifests", "copy-plan.json")
	copyManifestArg := copyManifest
	if runtime.GOOS == "windows" {
		copyManifestArg = filepath.ToSlash(copyManifestArg)
	}

	copyDestination := filepath.Join(dir, "copy-out")
	copyBase, err := filepath.Abs(filepath.Join("testdata", "07-cp"))
	if err != nil {
		t.Fatal(err)
	}
	copyDestinationArg := relFrom(copyBase, copyDestination)

	syncWorktree := filepath.Join(dir, "sync-worktree")
	if err := os.Mkdir(syncWorktree, 0755); err != nil {
		t.Fatal(err)
	}
	syncScratch := filepath.Join(dir, "sync-scratch")
	if err := os.Mkdir(syncScratch, 0755); err != nil {
		t.Fatal(err)
	}
	syncScratchFull := filepath.Join(dir, "sync-scratch-full")
	if err := os.Mkdir(syncScratchFull, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(syncScratchFull, "leftover.txt"), []byte("busy"), 0644); err != nil {
		t.Fatal(err)
	}
	syncState := filepath.Join(dir, "sync.state")
	if err := os.WriteFile(syncState, []byte("state"), 0644); err != nil {
		t.Fatal(err)
	}
	syncHelper := filepath.Join(dir, "sync-helper")
	if err := os.WriteFile(syncHelper, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	syncHelperPlain := filepath.Join(dir, "sync-helper.txt")
	if err := os.WriteFile(syncHelperPlain, []byte("helper"), 0644); err != nil {
		t.Fatal(err)
	}
	syncStateReadOnly := filepath.Join(dir, "sync-readonly.state")
	if err := os.WriteFile(syncStateReadOnly, []byte("state"), 0444); err != nil {
		t.Fatal(err)
	}
	syncCurrentTarget := filepath.Join(dir, "release-2026-04-18")
	if err := os.Mkdir(syncCurrentTarget, 0755); err != nil {
		t.Fatal(err)
	}
	syncCurrent := filepath.Join(dir, "current-release")
	if err := os.Symlink(syncCurrentTarget, syncCurrent); err != nil {
		t.Fatal(err)
	}
	syncCurrentFile := filepath.Join(dir, "current-release.txt")
	if err := os.WriteFile(syncCurrentFile, []byte("not a link"), 0644); err != nil {
		t.Fatal(err)
	}
	syncLockTaken := filepath.Join(dir, "sync.lock")
	if err := os.WriteFile(syncLockTaken, []byte("locked"), 0644); err != nil {
		t.Fatal(err)
	}
	outputParentFile := filepath.Join(dir, "output-parent")
	if err := os.WriteFile(outputParentFile, []byte("parent"), 0644); err != nil {
		t.Fatal(err)
	}
	outputReadOnlyDir := filepath.Join(dir, "output-read-only")
	if err := os.Mkdir(outputReadOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	unreadableCACert := filepath.Join(dir, "blocked.pem")
	if err := os.WriteFile(unreadableCACert, []byte("cert"), 0000); err != nil {
		t.Fatal(err)
	}
	cleanParent := filepath.Join(dir, "clean-parent")
	if err := os.Mkdir(cleanParent, 0755); err != nil {
		t.Fatal(err)
	}
	cleanTemplateTarget := filepath.Join(dir, "template.txt")
	if err := os.WriteFile(cleanTemplateTarget, []byte("template"), 0644); err != nil {
		t.Fatal(err)
	}
	cleanTemplate := filepath.Join(cleanParent, "..", "template.txt")

	t.Setenv("GOCOVERDIR", cov)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOCACHE", filepath.Join(dir, "gocache"))
	t.Setenv("GOTELEMETRY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")

	networkCtx, cancel := context.WithTimeout(context.Background(), 2e9)
	defer cancel()

	publicNetwork := true
	if _, err := net.DefaultResolver.LookupHost(networkCtx, "google.com"); err != nil {
		publicNetwork = false
	}

	examples := []string{
		"01-minimal",
		"02-simple",
		"03-commands",
		"04-docker",
		"05-curl",
		"06-git",
		"07-cp",
		"08-head",
		"09-percentile",
		"10-sync",
		"11-aws",
		"12-http",
	}

	for _, name := range examples {
		for _, args := range [][]string{
			{
				"get",
				"-tool",
				"github.com/lachaloupe/cli/cmd/cligen",
			},
			{
				"generate",
				".",
			},
			{
				"build",
				"-o", filepath.Join(dir, name),
				"-ldflags", "-X github.com/lachaloupe/cli.Version=1.2.3",
				"-cover",
				"-coverpkg=github.com/lachaloupe/cli,example.com/testcli",
				".",
			},
		} {
			cmd := exec.Command("go", args...)
			cmd.Dir = filepath.Join("testdata", name)
			cmd.Env = os.Environ()
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
			}
		}
	}

	tests := []struct {
		name     string
		binary   string
		args     []string
		env      []string
		check    func(*testing.T)
		skip     bool
		status   int
		expected string
	}{
		{
			name:   "help",
			binary: "01-minimal",
			args:   []string{"--help"},
		},
		{
			name:     "success",
			binary:   "01-minimal",
			args:     []string{"--some-flag", "hello"},
			expected: "{hello}\n",
		},
		{
			name:     "unknown-flag",
			binary:   "01-minimal",
			args:     []string{"--missing"},
			status:   1,
			expected: "error: unknown flag \"missing\"\n",
		},
		{
			name:     "missing-value",
			binary:   "01-minimal",
			args:     []string{"--some-flag"},
			status:   1,
			expected: "error: missing value for flag \"some-flag\"\n",
		},
		{
			name:   "version",
			binary: "01-minimal",
			args:   []string{"version"},
		},
		{
			name:   "help",
			binary: "02-simple",
			args:   []string{"--help"},
		},
		{
			name:     "success",
			binary:   "02-simple",
			args:     []string{"-i", "-r", "-m", "2", "demo"},
			expected: "{demo [] true false true 2}\n",
		},
		{
			name:     "invert-alias",
			binary:   "02-simple",
			args:     []string{"-v", "demo"},
			expected: "{demo [] false true false 10}\n",
		},
		{
			name:     "invert-alias-explicit-false",
			binary:   "02-simple",
			args:     []string{"-v", "false", "demo"},
			expected: "{demo [] false false false 10}\n",
		},
		{
			name:     "missing-required",
			binary:   "02-simple",
			status:   1,
			expected: "error: missing required argument \"pattern\"\n",
		},
		{
			name:     "bad-count",
			binary:   "02-simple",
			args:     []string{"-m", "nope", "demo"},
			status:   1,
			expected: "error: invalid value for argument \"max-count\": strconv.ParseUint: parsing \"nope\": invalid syntax\n",
		},
		{
			name:   "help",
			binary: "03-commands",
			args:   []string{"--help"},
		},
		{
			name:   "login",
			binary: "03-commands",
			args:   []string{"login", "--user", "alice", "--password", "secret"},
		},
		{
			name:   "signout-alias",
			binary: "03-commands",
			args:   []string{"signout"},
		},
		{
			name:     "unknown-subcommand",
			binary:   "03-commands",
			args:     []string{"wat"},
			status:   1,
			expected: "error: unexpected argument \"wat\"\n",
		},
		{
			name:   "help",
			binary: "04-docker",
			args:   []string{"--help"},
		},
		{
			name:   "container-run-help",
			binary: "04-docker",
			args:   []string{"container", "run", "--help"},
		},
		{
			name:   "root-flags",
			binary: "04-docker",
			args:   []string{"--config", "config.json"},
		},
		{
			name:     "network-create",
			binary:   "04-docker",
			args:     []string{"network", "create", "--gateway-mac", "02:42:ac:11:00:02", "--subnet", "10.0.0.0/24", "--ip-range", "10.0.0.0/25", "demo-net"},
			expected: "",
		},
		{
			name:     "root-default-host",
			binary:   "04-docker",
			args:     nil,
			expected: "docker api-version= config= debug=false host=unix:///var/run/docker.sock tls=false tlscacert=\n",
		},
		{
			name:     "bad-creatable",
			binary:   "04-docker",
			args:     []string{"--config", "missing/child"},
			status:   1,
			expected: "error: invalid value for argument \"config\": \"config\" must have an existing parent directory\n",
		},
		{
			name:     "bad-network-gateway-mac",
			binary:   "04-docker",
			args:     []string{"network", "create", "--gateway-mac", "not-a-mac", "demo-net"},
			status:   1,
			expected: "error: invalid value for argument \"gateway-mac\": address not-a-mac: invalid MAC address\n",
		},
		{
			name:     "bad-network-subnet",
			binary:   "04-docker",
			args:     []string{"network", "create", "--subnet", "10.0.0.0", "demo-net"},
			status:   1,
			expected: "error: invalid value for argument \"subnet\": invalid CIDR address: 10.0.0.0\n",
		},
		{
			name:   "container-run",
			binary: "04-docker",
			args: []string{
				"container", "run",
				"--detach",
				"--env", "APP_ENV=dev",
				"--publish", "8080:80",
				"--name", "web",
				"nginx:latest",
				"echo", "hello",
			},
		},
		{
			name:     "bad-pull",
			binary:   "04-docker",
			args:     []string{"container", "run", "--pull", "sometimes", "nginx:latest"},
			status:   1,
			expected: "error: invalid value for argument \"pull\": must be one of: always, missing, never\n",
		},
		{
			name:   "env-file",
			binary: "04-docker",
			args: []string{
				"container", "run",
				"--env-file", "Dockerfile.env",
				"nginx:latest",
			},
		},
		{
			name:     "bad-env-file",
			binary:   "04-docker",
			args:     []string{"container", "run", "--env-file", ".", "nginx:latest"},
			status:   1,
			expected: "error: invalid value for argument \"env-file\": \"env-file\" must reference a file\n",
		},
		{
			name:     "bad-clean-path",
			binary:   "04-docker",
			args:     []string{"--config", "../escape"},
			status:   1,
			expected: "error: invalid value for argument \"config\": \"config\" must not backtrack outside its root\n",
		},
		{
			name:   "image-tag",
			binary: "04-docker",
			args:   []string{"image", "tag", "alpine:3.20", "registry.example.com/demo/alpine:latest"},
		},
		{
			name:     "defaults",
			binary:   "05-curl",
			args:     []string{"https://example.com"},
			expected: "curl method=GET headers= data= location=false connect-timeout=5s retry=0 output= cacert= proxy= user-agent=curl/8.0 url=https://example.com\n",
		},
		{
			name:   "request",
			binary: "05-curl",
			args: []string{
				"-X", "POST",
				"-H", "Accept: application/json",
				"-H", "X-Debug: true",
				"-d", "name=demo",
				"-L",
				"--connect-timeout", "250ms",
				"--retry", "2",
				"--proxy", "https://proxy.example.com",
				"--user-agent", "curl/8.7.1",
				"https://example.com/api",
			},
			expected: "curl method=POST headers=Accept: application/json,X-Debug: true data=name=demo location=true connect-timeout=250ms retry=2 output= cacert= proxy=https://proxy.example.com user-agent=curl/8.7.1 url=https://example.com/api\n",
		},
		{
			name:     "headers-replace",
			binary:   "05-curl",
			args:     []string{"-H", `["Accept: application/json","X-Debug: true"]`, "https://example.com/api"},
			expected: "curl method=GET headers=Accept: application/json,X-Debug: true data= location=false connect-timeout=5s retry=0 output= cacert= proxy= user-agent=curl/8.0 url=https://example.com/api\n",
		},
		{
			name:     "data-from-file",
			binary:   "05-curl",
			args:     []string{"--data", "@payload.txt", "https://example.com/upload"},
			expected: "",
		},
		{
			name:     "bad-timeout",
			binary:   "05-curl",
			args:     []string{"--connect-timeout", "soon", "https://example.com"},
			status:   1,
			expected: "error: invalid value for argument \"connect-timeout\": time: invalid duration \"soon\"\n",
		},
		{
			name:     "location-alias-with-positional-before-error",
			binary:   "05-curl",
			args:     []string{"-L", "https://example.com", "--retry", "nope"},
			status:   1,
			expected: "error: invalid value for argument \"retry\": strconv.ParseUint: parsing \"nope\": invalid syntax\n",
		},
		{
			name:     "bad-cacert",
			binary:   "05-curl",
			args:     []string{"--cacert", ".", "https://example.com"},
			status:   1,
			expected: "error: invalid value for argument \"cacert\": \"cacert\" must reference a file\n",
		},
		{
			name:     "bad-output-parent-not-dir",
			binary:   "05-curl",
			args:     []string{"--output", filepath.Join(outputParentFile, "result.txt"), "https://example.com"},
			status:   1,
			expected: "error: invalid value for argument \"output\": \"output\" must have a directory parent\n",
		},
		{
			name:   "bad-output-parent-not-writeable",
			binary: "05-curl",
			args:   []string{"--output", filepath.Join(outputReadOnlyDir, "result.txt"), "https://example.com"},
			status: 1,
			skip:   runtime.GOOS == "windows",
		},
		{
			name:   "bad-cacert-not-readable",
			binary: "05-curl",
			args:   []string{"--cacert", unreadableCACert, "https://example.com"},
			status: 1,
			skip:   runtime.GOOS == "windows",
		},
		{
			name:   "commit-defaults",
			binary: "06-git",
			args:   []string{"commit"},
			env: []string{
				"GIT_MESSAGE=",
			},
			expected: "commit all=false amend=false cleanup=strip author= message=chore: bootstrap repository signoff=false scope= template= paths=\n",
		},
		{
			name:   "commit-help",
			binary: "06-git",
			args:   []string{"commit", "--help"},
		},
		{
			name:   "commit-env-primary",
			binary: "06-git",
			args:   []string{"commit", "-a", "--signoff=true", "README.md"},
			env: []string{
				"GIT_MESSAGE=feat: ship the docs",
			},
			expected: "commit all=true amend=false cleanup=strip author= message=feat: ship the docs signoff=true scope= template= paths=README.md\n",
		},
		{
			name:   "commit-optional-scope",
			binary: "06-git",
			args:   []string{"commit", "-m", "test"},
			env: []string{
				"GIT_SCOPE=feat",
			},
			expected: "commit all=false amend=false cleanup=strip author= message=test signoff=false scope=feat/commit template= paths=\n",
		},
		{
			name:     "bad-cleanup",
			binary:   "06-git",
			args:     []string{"commit", "--cleanup", "aggressive"},
			status:   1,
			expected: "error: invalid value for argument \"cleanup\": unknown cleanup mode \"aggressive\"\n",
		},
		{
			name:     "commit-literal-at",
			binary:   "06-git",
			args:     []string{"commit", "-m", "@@feat: add release notes"},
			expected: "commit all=false amend=false cleanup=strip author= message=@feat: add release notes signoff=false scope= template= paths=\n",
		},
		{
			name:     "remote-add",
			binary:   "06-git",
			args:     []string{"remote", "add", "-f=true", "origin", "https://example.com/demo/repo.git"},
			expected: "remote-add fetch=true name=origin url=https://example.com/demo/repo.git\n",
		},
		{
			name:     "bad-remote-url",
			binary:   "06-git",
			args:     []string{"remote", "add", "origin", "https://%zz"},
			status:   1,
			expected: "error: invalid value for argument \"url\": parse \"https://%zz\": invalid URL escape \"%zz\"\n",
		},
		{
			name:     "bad-author",
			binary:   "06-git",
			args:     []string{"commit", "--author", "not-an-address"},
			status:   1,
			expected: "error: invalid value for argument \"author\": mail: missing '@' or angle-addr\n",
		},
		{
			name:     "bad-template",
			binary:   "06-git",
			args:     []string{"commit", "--template", "."},
			status:   1,
			expected: "error: invalid value for argument \"template\": \"template\" must reference a file\n",
		},
		{
			name:     "template-clean-backtrack",
			binary:   "06-git",
			args:     []string{"commit", "--template", cleanTemplate, "-m", "template check"},
			expected: "commit all=false amend=false cleanup=strip author= message=template check signoff=false scope= template=" + cleanTemplate + " paths=\n",
		},
		{
			name:     "copy",
			binary:   "07-cp",
			args:     []string{"-a", "-R=true", "--manifest", copyManifestArg, "bin/*.txt", "docs/guide.txt", copyDestinationArg},
			expected: "cp archive=true force=false interactive=false recursive=true manifest=" + copyManifestArg + " sources=bin/*.txt,docs/guide.txt destination=" + copyDestinationArg + "\n",
			check: func(t *testing.T) {
				info, err := os.Stat(copyDestination)
				if err != nil {
					t.Fatalf("destination was not created: %v", err)
				}
				if !info.IsDir() {
					t.Fatal("destination was not created as a directory")
				}
			},
		},
		{
			name:     "dash-path",
			binary:   "07-cp",
			args:     []string{"--", "-draft.txt", "dest.txt"},
			expected: "cp archive=false force=false interactive=false recursive=false manifest= sources=-draft.txt destination=dest.txt\n",
		},
		{
			name:     "bad-manifest-relative",
			binary:   "07-cp",
			args:     []string{"--manifest", "copy-plan.json", "docs/*.txt", "dist/"},
			status:   1,
			expected: "error: invalid value for argument \"manifest\": \"manifest\" must reference an absolute path\n",
		},
		{
			name:     "bad-manifest-extension",
			binary:   "07-cp",
			args:     []string{"--manifest", filepath.Join(dir, "manifests", "copy-plan.txt"), "docs/*.txt", "dist/"},
			status:   1,
			expected: "error: invalid value for argument \"manifest\": \"manifest\" must use one of these extensions: [.json]\n",
		},
		{
			name:     "bad-source-abs",
			binary:   "07-cp",
			args:     []string{filepath.Join(dir, "docs", "*.txt"), "dist/"},
			status:   1,
			expected: "error: invalid value for argument \"sources\": \"sources\" must reference a relative path\n",
		},
		{
			name:     "bad-source-glob",
			binary:   "07-cp",
			args:     []string{"docs/[", "dist/"},
			status:   1,
			expected: "error: invalid value for argument \"sources\": \"sources\" must be a valid glob pattern: syntax error in pattern\n",
		},
		{
			name:     "bad-destination-abs",
			binary:   "07-cp",
			args:     []string{"docs/*.txt", filepath.Join(dir, "dist")},
			status:   1,
			expected: "error: invalid value for argument \"destination\": \"destination\" must reference a relative path\n",
		},
		{
			name:     "missing-destination",
			binary:   "07-cp",
			args:     nil,
			status:   1,
			expected: "error: missing required argument \"destination\"\n",
		},
		{
			name:     "lines",
			binary:   "08-head",
			args:     []string{"-n", "5", "-v=true", "app.log", "worker.log"},
			expected: "",
		},
		{
			name:     "default-lines",
			binary:   "08-head",
			args:     []string{"README.md"},
			expected: "",
		},
		{
			name:     "default-files",
			binary:   "08-head",
			args:     nil,
			expected: "",
		},
		{
			name:     "bad-lines",
			binary:   "08-head",
			args:     []string{"-n", "ten"},
			status:   1,
			expected: "error: invalid value for argument \"lines\": strconv.ParseUint: parsing \"ten\": invalid syntax\n",
		},
		{
			name:   "help",
			binary: "09-percentile",
			args:   []string{"--help"},
		},
		{
			name:     "defaults",
			binary:   "09-percentile",
			args:     []string{"12.4", "15.8", "20.1", "9.7"},
			expected: "percentile scale=1 percentiles=50,95,99 samples=9.7,12.4,15.8,20.1 p50=12.4 p95=20.1 p99=20.1\n",
		},
		{
			name:     "scaled-custom-percentiles",
			binary:   "09-percentile",
			args:     []string{"--scale", "1000", "--percentiles", "[90,99]", "0.12", "0.15", "0.18", "0.3"},
			expected: "percentile scale=1000 percentiles=90,99 samples=120,150,180,300 p90=300 p99=300\n",
		},
		{
			name:     "float32-scale",
			binary:   "09-percentile",
			args:     []string{"--scale", "1.25", "1.5", "2"},
			expected: "percentile scale=1.25 percentiles=50,95,99 samples=1.875,2.5 p50=1.875 p95=2.5 p99=2.5\n",
		},
		{
			name:     "clear-then-append",
			binary:   "09-percentile",
			args:     []string{"--percentiles", "[]", "--percentiles", "90", "12.4", "15.8", "20.1", "9.7"},
			expected: "percentile scale=1 percentiles=90 samples=9.7,12.4,15.8,20.1 p90=20.1\n",
		},
		{
			name:     "bad-scale",
			binary:   "09-percentile",
			args:     []string{"--scale", "fast", "12.4"},
			status:   1,
			expected: "error: invalid value for argument \"scale\": strconv.ParseFloat: parsing \"fast\": invalid syntax\n",
		},
		{
			name:     "success",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--workers", "8", "--retries", "3", "--priority", "-2", "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			expected: "sync worktree=" + syncWorktree + " workers=8 retries=3 priority=-2 scratch=" + syncScratch + " state=" + syncState + " helper=" + syncHelper + " lock=" + filepath.Join(dir, "next-sync.lock") + " current=" + syncCurrent + " sources=assets/*.txt destination=public/\n",
		},
		{
			name:     "bad-worktree-missing",
			binary:   "10-sync",
			args:     []string{"--worktree", filepath.Join(dir, "missing-worktree"), "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"worktree\": \"worktree\" must reference an existing path\n",
		},
		{
			name:     "bad-worktree-file",
			binary:   "10-sync",
			args:     []string{"--worktree", syncState, "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"worktree\": \"worktree\" must reference a directory\n",
		},
		{
			name:     "bad-scratch-missing",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", filepath.Join(dir, "missing-scratch"), "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"scratch\": \"scratch\" must reference an existing path\n",
		},
		{
			name:     "bad-scratch-file",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncState, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"scratch\": \"scratch\" must reference a directory\n",
		},
		{
			name:     "bad-scratch-not-empty",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratchFull, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"scratch\": \"scratch\" must reference an empty directory\n",
		},
		{
			name:     "bad-state-missing",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", filepath.Join(dir, "missing.state"), "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"state\": \"state\" must reference an existing path\n",
		},
		{
			name:   "bad-state-not-writeable",
			binary: "10-sync",
			args:   []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncStateReadOnly, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status: 1,
			skip:   runtime.GOOS == "windows",
		},
		{
			name:     "bad-helper-missing",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncState, "--helper", filepath.Join(dir, "missing-helper"), "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"helper\": \"helper\" must reference an existing path\n",
		},
		{
			name:     "bad-helper-not-exec",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncState, "--helper", syncHelperPlain, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"helper\": \"helper\" must reference an executable path\n",
		},
		{
			name:     "bad-lock-exists",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", syncLockTaken, "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"lock\": \"lock\" must not reference an existing path\n",
		},
		{
			name:     "bad-current-missing",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", filepath.Join(dir, "missing-current"), "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"current\": \"current\" must reference an existing path\n",
		},
		{
			name:     "bad-current-regular",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrentFile, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"current\": \"current\" must reference a symlink\n",
		},
		{
			name:     "bad-workers",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--workers", "many", "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"workers\": strconv.ParseInt: parsing \"many\": invalid syntax\n",
		},
		{
			name:     "bad-retries",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--retries", "300", "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"retries\": strconv.ParseUint: parsing \"300\": value out of range\n",
		},
		{
			name:     "bad-priority",
			binary:   "10-sync",
			args:     []string{"--worktree", syncWorktree, "--priority", "200", "--scratch", syncScratch, "--state", syncState, "--helper", syncHelper, "--lock", filepath.Join(dir, "next-sync.lock"), "--current", syncCurrent, "assets/*.txt", "public/"},
			status:   1,
			expected: "error: invalid value for argument \"priority\": strconv.ParseInt: parsing \"200\": value out of range\n",
		},
		{
			name:   "http-and-s3",
			binary: "11-aws",
			args:   []string{"https://jsonplaceholder.typicode.com/todos/1/", "s3://ont-open-data/gm24385_2020.09/README.md"},
			env:    []string{"AWS_EC2_METADATA_DISABLED=true", "AWS_REGION=eu-west-1"},
			skip:   !publicNetwork,
		},
		{
			name:     "bad-s3-uri",
			binary:   "11-aws",
			args:     []string{"s3://ont-open-data"},
			status:   1,
			expected: "error: invalid value for argument \"files\": files: invalid S3 URI \"s3://ont-open-data\"\n",
		},
		{
			name:   "help",
			binary: "12-http",
			args:   []string{"--help"},
		},
		{
			name:   "serve-help",
			binary: "12-http",
			args:   []string{"serve", "--help"},
		},
	}

	for _, test := range tests {
		t.Run(test.binary+"/"+test.name, func(t *testing.T) {
			if test.skip {
				t.Skip("skipped on this platform")
			}

			cmd := exec.Command(filepath.Join(dir, test.binary), test.args...)
			cmd.Dir = filepath.Join("testdata", test.binary)
			cmd.Env = append(os.Environ(), test.env...)

			out, err := cmd.CombinedOutput()

			if exit, ok := err.(*exec.ExitError); ok {
				if exit.ExitCode() != test.status {
					t.Fatalf("got status %d; want %d\n%s", exit.ExitCode(), test.status, out)
				}
			} else {
				if test.status != 0 {
					t.Fatalf("got no error; want status %d\n%s", test.status, out)
				}
			}

			expected := test.expected
			goldenPath := ""

			if expected != "" && strings.Count(expected, "\n") > 1 {
				t.Fatalf("inline expected output must be single-line: %q", test.name)
			}

			if expected == "" {
				goldenPath = filepath.Join("testdata", "expected", test.binary+"+"+test.name+".txt")
				data, err := os.ReadFile(goldenPath)
				if err != nil {
					if !updateGolden || !os.IsNotExist(err) {
						t.Fatal(err)
					}
				} else {
					expected = string(data)
				}
			}

			if test.binary == "01-minimal" && test.name == "version" {
				expected = strings.ReplaceAll(expected, "darwin/arm64", runtime.GOOS+"/"+runtime.GOARCH)
			}

			got := string(out)
			if goldenPath != "" && got != expected && updateGolden {
				if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
					t.Fatal(err)
				}
				expected = got
			}

			if got != expected {
				t.Fatalf("got output:\n%s\nwant:\n%s", got, expected)
			}

			if test.check != nil {
				test.check(t)
			}
		})
	}

	files, err := os.ReadDir(cov)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatalf("no coverage data files were emitted to %s", cov)
	}

	cmd := exec.Command("go", "tool", "covdata", "percent", "-i="+cov)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool covdata percent: %v\n%s", err, out)
	}

	lines := []string{}

	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 5 && fields[0] == "github.com/lachaloupe/cli" {
			lines = append(lines, strings.Join(fields, " "))
		}
	}

	if len(lines) != 1 {
		t.Fatal("unexpected number of coverage data lines")
	}

	t.Log(lines[0])
}
