# cli

Generate command-line interfaces from Go code.

## Install

```bash
# add the runtime package to your project
go get github.com/lachaloupe/cli

# add the generator as a project-local tool
go get -tool github.com/lachaloupe/cli/cmd/cligen
```

## Quick start

Here's a minimal complete working example:

```go
package main

import (
	"context"

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
	return nil
}

func main() {
	CLI.Main()
}
```

```bash
# build it
go generate ./...
go build

# run it
app --name alice
```

## Features

### 1. Flags are inferred from struct fields used by the handler

```go
type Args struct {
	OutputFile string
	DryRun     bool
}

func Run(ctx context.Context, args Args) error {
	...
}
```

This becomes:

```bash
app --output-file result.txt --dry-run
```

- Field names are converted to kebab-case automatically.
- Command line is parsed and strings are converted to the field's type.

### 2. Help text comes from comments

```go
type Args struct {
	// Write to this file.
	Output string
}

// Convert one file into another.
func Run(ctx context.Context, args Args) error {
	...
}
```

Will extract comments from the handler and from fields:

```
Usage: app

Convert one file into another.

Options:
  --output         Write to this file. (string)
```

### 3. Positional arguments are explicit

```go
type Args struct {
	//cli:arg
	Pattern string

	//cli:arg
	Sources []string

	//cli:arg=2
	Position []float
}
```

This accepts:

```bash
app error ./cmd ./internal ./local 1.0 2.0
```

- `//cli:arg` with non-slice type consumes one value.
- `//cli:arg` with slice type consumes the remaining values unless specified.

### 4. Required arguments are declared in code

```go
type Args struct {
	//cli:required
	//cli:arg
	Pattern string
}
```

### 5. Defaults are declared next to the field

```go
type Args struct {
	//cli:default=10
	MaxCount uint
}
```

This behaves as if the user passed:

```bash
app --max-count 10
```

Default values also support basic environment expansion.
Multiple `//cli:default=` directives can be provided.
The first expanded value that is not empty is used as the default.
For example:

```go
type Args struct {
	//cli:default=$XDG_DATA_HOME/my-app
	//cli:default=$HOME/.local/share/my-app
	DataDir string
}
```

And it's also possible to supply a `New` constructor.

```go
var CLI = cli.Command{
	Handler: Run,
	New:     NewArgs,
}

func NewArgs() Args {
	return Args{
		Format: "json",
	}
}
```

- The constructor `New` is called first.
- Then, any default value directive is applied.
- And finally, parsed flags supplied by the user are applied last.

### 6. Short and alternate flag names are supported

```go
type Args struct {
	//cli:alias=i
	//cli:alias=ignore
	IgnoreCase bool

	//cli:alias=m
	MaxCount uint
}
```

This accepts:

```bash
app --ignore -m 5
```

### 7. Native value resolvers can expand files and provider-backed values

Any raw CLI value can be resolved before it is parsed into the target Go type.

By default, a value starting with `@` reads the content of a file:

```bash
app --config @./config.json
app --token @/run/secrets/api-token
```

This works for flags, positional arguments, and `//cli:default=` values.
The resolved text is then parsed as if the user had typed it directly.

To pass a literal value starting with `@`, escape it with another `@`:

```bash
app --message @@hello
```

This behaves as if the user passed:

```bash
app --message @hello
```

Provider-backed native values can be generated too.
For AWS support, invoke `cligen` with `--provider aws`:

```go
//go:generate go tool cligen --provider aws
```

That enables:

```bash
app --db-url @aws:ssm:/my-app/db-url
app --api-key @aws:secret:my-app/api-key
```

The generated code imports the AWS SDK directly, so the consuming module must add those dependencies itself.

### 8. Path constraints can be declared next to string arguments

```go
type Args struct {
	//cli:path=exists
	//cli:path=file
	Input string

	//cli:path=mkdir
	//cli:path=empty
	Scratch string
}
```

This accepts existing files for `Input`, and ensures `Scratch` exists as an empty directory.
The supported path directives are:

- `//cli:path=exists`
- `//cli:path=not-exists`
- `//cli:path=dir`
- `//cli:path=file`
- `//cli:path=mkdir`
- `//cli:path=creatable`
- `//cli:path=readable`
- `//cli:path=writeable`
- `//cli:path=symlink`
- `//cli:path=abs`
- `//cli:path=rel`
- `//cli:path=exec`
- `//cli:path=clean`
- `//cli:path=empty`
- `//cli:path=glob`
- `//cli:path=.ext`

Multiple `//cli:path=.ext` directives are allowed and are treated as OR checks.
These directives only apply to `string` and `[]string` fields, and they can be combined.
Under the hood, generated code attaches labels to the argument and wires `cli.PathValidate` into the generic per-argument validation hook.

### 9. Commands are declared as a tree

```go
var CLI = cli.Command{
	Commands: []*cli.Command{
		{
			Name: "login",
			Handler: RunLogin,
		},
		{
			Name: "logout",
			Handler: RunLogout,
		},
	},
}
```

This accepts:

```bash
app login --user alice
app logout
```

### 10. Commands can have aliases

```go
//cli:alias=signout
func RunLogout(ctx context.Context) error {
	...
}
```

### 11. Handlers may or may not take an argument struct

Without args:

```go
func RunLogout(ctx context.Context) error {
	...
}
```

With args:

```go
func RunLogin(ctx context.Context, args LoginArgs) error {
	...
}
```

### 14. Native types and `encoding.TextUnmarshaler`

Native scalar Go types are parsed automatically.
Slices are supported too.
For other types, the parser assumes the value implements `encoding.TextUnmarshaler`.
This includes common standard-library types such as:

- `time.Time`
- `net.IP`
- `netip.Addr`
- `netip.AddrPort`
- `netip.Prefix`
- `regexp.Regexp`
- `big.Int`
- `big.Rat`
- `big.Float`
- `x509.OID`
- `slog.Level`
- `slog.LevelVar`

Example:

```go
type Args struct {
	Timeout  time.Duration
	Origin   url.URL
	Subnet   net.IPNet
	MAC      net.HardwareAddr
	From     mail.Address
	When     time.Time
	Resolver netip.Addr
	Pattern  regexp.Regexp
}
```

```bash
app \
  --timeout 500ms \
  --origin https://example.com/api \
  --subnet 192.0.2.1/24 \
  --mac 01:23:45:67:89:ab \
  --from 'Alice <alice@example.com>' \
  --when 2026-04-05T12:00:00Z \
  --resolver 1.1.1.1 \
  --pattern '^demo$'
```

`io.Reader` is also supported as a native field type.
Its value is treated as:

- a filename to open
- `-` to use `stdin`
- a `file://`, `http://`, or `https://` URI
- an `s3://` URI when code is generated with `--provider aws`

Example:

```go
type Args struct {
	Input io.Reader
}
```

```bash
app --input payload.json
app --input -
app --input file:///tmp/payload.json
app --input https://example.com/payload.json
app --input s3://my-bucket/payload.json
```

### 15. The runtime parser handles common CLI forms

Supported forms include:

```bash
app --name alice
app --name=alice
app -v
app -v=false
app -- -1 -2
```

`--` stops flag parsing, which is useful for negative positional values.

## Examples

There are more examples [here](./cmd/cligen/testdata).

## How it works

- The `cli.Command` variable is parsed by `cligen` when invoked by `go generate`
- It finds the specified `Handler`
- The `struct` parameter of that function is used to infer CLI arguments
- An helper function is generated in `$GOFILE.cli.go`
- You have your CLI!
- At runtime, the `Main` function calls that helper function
- The `cli` parser fills structs from `os.Args` and calls your `Handler`
