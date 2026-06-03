# Domain

## Goals

- Go sources (handlers/structs/fields/comments) as the source of truth for CLI surfaces.
- Reduce the burden of implementing common CLI patterns.
- No magic, no runtime dependencies, no manual duplication.

## Glossary

- **cligen**: code generator that reads a `cli.Command` tree and emits `*.cli.go` glue.
- **runtime**: the `github.com/lachaloupe/cli` library consumed by generated code.
- **directive**: `//cli:` comment annotations on struct fields that shape generated CLI behavior.
- **hook**: runtime callbacks (`Context`, `New`, `Lookup`, `Open`) that customize lifecycle.
- **provider**: opt-in generation flavor (e.g. `--provider aws`) that installs hooks and adds dependencies on the consumer side.

## Architecture

- Struct fields become flags/positionals; comments become help text.
- Directives and comments cover common use-cases; hooks and explicit fields on `cli.Command`/`cli.Arg` are the escape hatch and always take precedence over generated values.
- Runtime has zero external dependencies; providers add dependencies on the CLI user's side.
- Command trees can span multiple packages.
- Panics for programmatic misuses; errors for user input.

## Testing

- End-to-end golden tests (`cmd/cligen/testdata`) are the primary coverage tool.
- Unit tests only where isolation is needed.

## Invariants

- `Parse` mutates the command tree in place.
  Callers needing independent parses must build a fresh tree.

## Parser Behavior

- Bool flags are implicitly `true` when specified without a value.
  If the next token parses as `strconv.ParseBool` (e.g. `0`, `1`, `true`, `false`) it is consumed as the flag value.
  Tokens starting with `-` are never consumed — `--flag=value` is the reliable disambiguator.
- `--[no-]flag`: bool flags automatically support negation.
  Conflict detection prevents user-defined args named `no-X` when `X` is a bool.
- Positionals are consumed before subcommand matching.
  The generator rejects positional arguments that are not required when subcommands are present.

## Known Issues

- Renamed import (`mycli "github.com/lachaloupe/cli"`) not detected — generator hardcodes `pkg.Name != "cli"`.
- Help text from doc comments includes function name prefix (Go convention).
  Use explicit `Help` field to override.
