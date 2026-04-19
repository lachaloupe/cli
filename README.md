# cli

[![Go Reference](https://pkg.go.dev/badge/github.com/lachaloupe/cli.svg)](https://pkg.go.dev/github.com/lachaloupe/cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/lachaloupe/cli)](https://goreportcard.com/report/github.com/lachaloupe/cli)
[![CI](https://github.com/lachaloupe/cli/actions/workflows/test.yml/badge.svg)](https://github.com/lachaloupe/cli/actions/workflows/test.yml)

Generate command-line interfaces from Go code.

`cligen` reads a `cli.Command`, infers the CLI surface from your handler signature, and writes the generated glue into `*.cli.go`.

## Install

```bash
# runtime dependency used by the generated CLI
go get github.com/lachaloupe/cli@v0.1.0

# tool-only dependency used by go generate
go get -tool github.com/lachaloupe/cli/cmd/cligen@v0.1.0
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: Run,
}

type Args struct {
	Name string
}

func Run(ctx context.Context, args Args) error {
	fmt.Println("hello", args.Name)
	return nil
}

func main() {
	CLI.Main()
}
```

```bash
# generate app.cli.go
go generate ./...

# build the binary
go build

# run it
./app --name alice
```

See [01-minimal](./cmd/cligen/testdata/01-minimal).

## How it works

Start with a `cli.Command` and a handler. If the handler takes an argument struct, `cligen` inspects that struct and turns its fields into flags or positional arguments. Comments become help text. Then `CLI.Main()` runs the generated parser and calls your handler.

## From Go fields to CLI UX

This:

```go
type Args struct {
	// Pattern to search for.
	//cli:required
	//cli:arg
	Pattern string

	// Files or directories to search in.
	//cli:arg
	Paths []string

	// Match case-insensitively.
	//cli:alias=i
	IgnoreCase bool

	// Stop after this many matches.
	//cli:alias=m
	//cli:default=10
	MaxCount uint
}
```

becomes a CLI like:

```bash
# pattern and paths are positional
./app error ./cmd ./internal

# flags come from the other fields
./app error . -i -m 25

# generated help reflects comments and defaults
./app --help
```

This is the core model:

- any struct fields become CLI typed inputs (not just exported ones)
- field names become kebab-case flags
- fields can be made positional by using the `//cli:arg` directive
- comments become help text

See [02-simple](./cmd/cligen/testdata/02-simple).

## Directives

These directives shape the generated CLI.

### Core directives

| Directive | Meaning |
| --- | --- |
| `//cli:arg` | Make the field positional. A scalar consumes one value. A slice consumes the remaining values. |
| `//cli:arg=N` | Make the field positional and consume exactly `N` values. |
| `//cli:arg=-1` | For slice fields, consume all remaining positional values. |
| `//cli:required` | Require the argument to be present. |
| `//cli:alias=x` | Add a short or alternate name. On command handlers, it adds a command alias. |
| `//cli:default=value` | Set a default value. Multiple defaults are allowed and are tried in order. |
| `//cli:enum` | Discover enum choices from `Strings() []string`. |
| `//cli:enum=value` | Add an allowed enum choice. |
| `//cli:path=...` | Apply path validation to `string` and `[]string` fields. |

### Path directives

| Directive | Meaning |
| --- | --- |
| `//cli:path=exists` | Path must exist. |
| `//cli:path=not-exists` | Path must not exist. |
| `//cli:path=dir` | Path must be a directory. |
| `//cli:path=file` | Path must be a regular file. |
| `//cli:path=mkdir` | Create the directory before validation. |
| `//cli:path=creatable` | Parent directory must exist and be writable. |
| `//cli:path=readable` | Path must be readable. |
| `//cli:path=writeable` | Path must be writable. |
| `//cli:path=symlink` | Path must be a symlink. |
| `//cli:path=abs` | Path must be absolute. |
| `//cli:path=rel` | Path must be relative. |
| `//cli:path=exec` | Path must be executable. |
| `//cli:path=clean` | Path must be lexically clean. |
| `//cli:path=empty` | Path must be an empty directory. |
| `//cli:path=glob` | Value must be a valid glob pattern. |
| `//cli:path=.ext` | Path must use one of the allowed file extensions. |

See [07-cp](./cmd/cligen/testdata/07-cp).

## Enums

Enum handling is opt-in.

- add `//cli:enum` to discover choices from `Strings() []string`
- non-native types can supply choices by implementing `Strings() []string`
- add one or more `//cli:enum=value` directives to define or extend the choice list

This works well with named types that already implement `encoding.TextUnmarshaler`.

```go
type Mode string

func (Mode) Strings() []string { return []string{"fast", "safe"} }

func (m *Mode) UnmarshalText(text []byte) error { ... }

type Args struct {
	//cli:enum
	Mode Mode

	//cli:enum=text
	//cli:enum=json
	Format string
}
```

See [06-git](./cmd/cligen/testdata/06-git).

## Native type parsing

Built-in scalar types are parsed automatically, and so are slices of those types. `encoding.TextUnmarshaler` types are supported too, which makes common standard-library types work out of the box.

Examples include:

- `time.Duration`
- `url.URL`
- `mail.Address`
- `net.IPNet`
- `net.HardwareAddr`

```bash
./app --timeout 5s --proxy http://proxy.internal:8080 --url https://example.com/upload
```

See [05-curl](./cmd/cligen/testdata/05-curl).

## Slices

Slices are first-class CLI inputs.

- slice flags append one value per occurrence
- slice positional arguments can consume the remaining values
- slice defaults use CSV syntax

```bash
./app --percentiles 90 --percentiles 99 120 150 180 300
```

See [09-percentile](./cmd/cligen/testdata/09-percentile).

## Parser behavior

Common forms are supported:

```bash
./app --name alice
./app --name=alice
./app -v
./app -v=false
./app -- -1 -2
```

`--` stops flag parsing, which is useful when positional values start with `-`.

## Commands and handler shapes

Commands are declared as a tree. A handler may take only `context.Context`, or `context.Context` plus an args struct.

```go
var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name:    "login",
			Handler: RunLogin,
		},
		{
			Name:    "logout",
			Handler: RunLogout,
		},
	},
}

func RunLogin(ctx context.Context, args LoginArgs) error { ... }
func RunLogout(ctx context.Context) error { ... }
```

That gives you a command tree such as:

```bash
./app login --user alice --password secret
./app logout
```

Parent command flags stay available under subcommands, which lets you define global flags once at the root and reuse them across the tree.

See [03-commands](./cmd/cligen/testdata/03-commands).

## Context values

Handlers always receive a `context.Context`. The generated code uses it for two things:

- `cli.Parent{}` stores the current command path
- `cli.Args(path)` stores the parsed args for a command path

That means a subcommand can read its own args or reach back to parent or root args when it needs shared configuration.

```go
rootArgs := ctx.Value(cli.Args("/")).(RootArgs)
loginArgs := ctx.Value(cli.Args("/login")).(LoginArgs)
```

Any command path works here, not just `/`. Use the path for the command whose parsed args you want to access.

See [03-commands](./cmd/cligen/testdata/03-commands).

## Defaults

Defaults live next to the field.

You can provide more than one `//cli:default=...` directive. They are evaluated in order, and the first one that expands to a non-empty value is used. This is mainly useful for environment-based fallbacks.

For example:

```go
//cli:default=$XDG_DATA_HOME/my-app
//cli:default=$HOME/.local/share/my-app
DataDir string
```

If `XDG_DATA_HOME` is set and not empty, it wins. Otherwise `HOME` is tried next. If neither expands to a non-empty value, no default is applied.

Slice defaults use CSV syntax:

```go
//cli:default=50,95,99
Percentiles []float64
```

At the command line:

```bash
# append values
./app --percentiles 90 --percentiles 99

# replace the whole slice
./app --percentiles "[90,99]"

# clear the slice
./app --percentiles "[]"
```

See [09-percentile](./cmd/cligen/testdata/09-percentile).

## Value resolution

Before parsing, raw values can be resolved. By default:

- `@path` reads from a file
- `//cli:default=$NAME` expands an environment variable

```go
//cli:default=$GIT_MESSAGE
//cli:default=@COMMIT_EDITMSG
Message string
```

This makes flows like these possible:

```bash
# read the request body from a file
./app --data @payload.txt https://example.com/upload

# use env first, then fall back to a file
./app commit
```

Note: use `@@value` to pass a literal leading `@`.

See [06-git](./cmd/cligen/testdata/06-git).

## `io.Reader` inputs

`io.Reader` is a native argument type. Values can point to:

- a local file
- `-` for stdin
- `file://`, `http://`, or `https://` URLs
- `s3://` URLs when generated with the AWS provider

```bash
./app app.log worker.log
./app -
./app https://example.com/file.txt
```

See [08-head](./cmd/cligen/testdata/08-head).

## AWS provider

Generate provider-aware code with:

```go
//go:generate go tool cligen --provider aws
```

That enables AWS-backed native values such as:

```bash
./app @aws:ssm:/my-app/config
./app @aws:secret:my-app/api-key
./app s3://my-bucket/object.txt
```

The generated code imports AWS SDK packages, so the consuming module must add those dependencies.

See [11-aws](./cmd/cligen/testdata/11-aws).

## Runtime hooks

The generated CLI is still just a `cli.Command`, so you can customize runtime behavior when needed.

| Hook | Purpose |
| --- | --- |
| `New` | Build the initial args value before defaults and user input are applied. |
| `Lookup` | Rewrite raw string values before built-in resolution and parsing. |
| `Open` | Rewrite raw values for `io.Reader` fields before file or URL resolution. |

### `New`

`New` seeds the initial args value before defaults and user input are applied.

```go
var CLI = cli.Command{
	Handler: RunHead,
	New:     NewArgs,
}

func NewArgs() Args {
	return Args{
		Files: []io.Reader{os.Stdin},
	}
}
```

In [08-head](./cmd/cligen/testdata/08-head), this makes standard input the default when no files are passed.

### `Lookup`

`Lookup` rewrites raw string values before built-in parsing runs. In practice, this is a string value provider hook.

The built-in resolver already handles `@path` file reads. `Lookup` exists for cases where the raw value should come from somewhere else before it is parsed into the target Go type.

The main shipped example is the AWS provider. When you generate with `--provider aws`, `cligen` installs a resolver under the hood so values like these are turned into plain strings before parsing:

```bash
./app --db-url @aws:ssm:/my-app/db-url
./app --api-key @aws:secret:my-app/api-key
```

See [11-aws](./cmd/cligen/testdata/11-aws).

### `Open`

`Open` rewrites raw values for `io.Reader` fields before built-in reader handling runs. In practice, this is a reader provider hook.

By default, `io.Reader` already understands:

- `-`
- local file paths
- `file://`
- `http://`
- `https://`

`Open` exists for cases where a value should produce a reader through another transport or container format first, such as S3 objects, compressed inputs, or archive entries.

The main shipped example is again the AWS provider. When generated with `--provider aws`, `cligen` installs a reader resolver so this works:

```bash
./app s3://my-bucket/object.txt
```

See [11-aws](./cmd/cligen/testdata/11-aws).

### `Arg.Parse`

`Arg.Parse` is a lower-level runtime hook for manual `cli.Command` definitions. Use it when a value should parse into a custom type or syntax that the built-in parser does not know about.

For example, to parse a date in `YYYY-MM-DD` form:

```go
cmd := cli.Command{
	Args: []*cli.Arg{
		{
			Name: "day",
			Type: "time.Time",
			Parse: func(s string) (any, error) {
				return time.Parse("2006-01-02", s)
			},
		},
	},
}
```

### `Arg.Validate`

`Arg.Validate` is the matching lower-level validation hook. It runs after parsing and is useful for domain checks that go beyond type conversion.

For example, to require an AWS ARN shape:

```go
cmd := cli.Command{
	Args: []*cli.Arg{
		{
			Name: "role-arn",
			Type: "string",
			Validate: func(arg *cli.Arg, s string) error {
				if !strings.HasPrefix(s, "arn:aws:") {
					return fmt.Errorf("%s must be an AWS ARN", arg.Name)
				}
				return nil
			},
		},
	},
}
```

## Built-in version command

If `cli.Version` is set, generated CLIs add a root `version` command automatically.

It is meant to be injected at build time:

```bash
go build -ldflags="-X github.com/lachaloupe/cli.Version=v1.2.3"
```

Then:

```bash
./app version
```

prints:

```text
go1.26.2 darwin/arm64
v1.2.3
```

If `cli.Version` is empty, no built-in version command is added. If your CLI already defines its own `version` command, that one is kept.

See [01-minimal](./cmd/cligen/testdata/01-minimal).

## Example suite

The examples under [cmd/cligen/testdata](./cmd/cligen/testdata) are also golden tests. Their checked-in `main.cli.go` files must match freshly generated output.

| Example | What it showcases |
| --- | --- |
| [01-minimal](./cmd/cligen/testdata/01-minimal) | Smallest generated CLI, help text, built-in version command |
| [02-simple](./cmd/cligen/testdata/02-simple) | Positional arguments, required values, aliases, defaults |
| [03-commands](./cmd/cligen/testdata/03-commands) | Subcommands, command aliases, handlers with and without args |
| [04-docker](./cmd/cligen/testdata/04-docker) | Larger command tree, inherited root flags, fixed and trailing positionals |
| [05-curl](./cmd/cligen/testdata/05-curl) | Native type parsing, durations, URLs, path validation |
| [06-git](./cmd/cligen/testdata/06-git) | Value resolution from env and files, command nesting |
| [07-cp](./cmd/cligen/testdata/07-cp) | Path directives, globs, relative and absolute path constraints |
| [08-head](./cmd/cligen/testdata/08-head) | `New`, `io.Reader`, stdin defaults |
| [09-percentile](./cmd/cligen/testdata/09-percentile) | Slice flags, slice defaults, numeric parsing |
| [10-sync](./cmd/cligen/testdata/10-sync) | Rich path validation across many path labels |
| [11-aws](./cmd/cligen/testdata/11-aws) | AWS provider, S3-backed readers |

Run the end-to-end example suite with:

```bash
cd cmd/cligen
go test -run TestUsingExamples -v
```
