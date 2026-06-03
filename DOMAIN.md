# Domain

## Glossary

- **cligen**: code generator that reads a `cli.Command` tree and emits `*.cli.go` glue
- **runtime**: the `github.com/lachaloupe/cli` library consumed by generated code
- **directive**: `//cli:` comment annotations on struct fields that shape generated CLI behavior
- **hook**: runtime callbacks (`Context`, `New`, `Lookup`, `Open`) on `Command` that customize lifecycle
- **provider**: opt-in code generation flavor (e.g. `--provider aws`) that installs hooks

## Architecture

- Generator emits `init()` that calls `CLI.Register(invokeCLI)` — overlays onto user's live `Command` struct.
- Runtime has zero external dependencies.
- Generation targets the source package. Cross-package support is handlers only, not command trees.
- Panics for programmer errors (unsupported type, nil invoke, bad template), errors for user input.

## Invariants

- `//cli:path=` conflict detection is exhaustive in `applyPathDirective` (generator-side only, not runtime).
- `Cleanup()` runs after handler via `errors.Join(handler(), cmd.Cleanup())` and in `Main` on error paths.
- Multi-default evaluation: first non-empty expansion wins, strict vs optional controls skip behavior.
- `Arg.Default` and `Arg.Defaults` are mutually exclusive at runtime — help.go panics if both are set.
- `io.Reader` fields auto-register cleanup: `openReader` appends `closer.Close` to `cmd.Cleanups` for any reader implementing `io.Closer`.
- `Parse` mutates the command tree in place (stores parsed values and cleanup handlers). Callers needing independent parses must build a fresh tree.

## Constraints

- Single output file per `go generate` invocation.
- No package load caching — each cross-package handler reference loads the package independently.
- Template rendering is all-or-nothing per command (no partials/composition).
- Command paths are flat strings (`"/image/ls"`) — no compile-time safety when referencing parent args.
- Non-leaf commands with subcommands auto-return `ErrHelp` when no subcommand is selected (generated behavior).

## Known Issues

- Help text from doc comments includes function name prefix (Go convention).
  Use explicit `Help` field to override.
- Renamed import (`mycli "github.com/lachaloupe/cli"`) not detected by generator — hardcodes `pkg.Name != "cli"`.

## Parser Behavior

- Bool flags: if the next token parses via `strconv.ParseBool`, it is consumed. Otherwise the flag defaults to `true`.
- Tokens starting with `-` are never consumed as bool values — the flag defaults to `true` instead.
- `--flag=value` is the reliable disambiguator for bools.
- `--[no-]flag`: bool flags automatically support `--no-X` to set the flag to `false`. Help renders as `--[no-]X`.
- Conflict detection prevents user-defined args named `no-X` when `X` is a bool.

## Assumptions

- Single-file generation is acceptable long-term since multi-package support is supported.

## Unresolved Questions

(none)

## Rejected Ideas

(none yet)
