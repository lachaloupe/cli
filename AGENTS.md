# Releases

* Update `go.mod` in `cmd/cligen` so it references the next release version.
* Use the latest installed version of Go in all `go.mod` files.
* Update all direct dependencies to their latest versions and run `go mod tidy`.
* Regenerate `cmd/cligen/main.cli.go` with `go generate ./cmd/cligen`.
* Run tests with `UPDATE_GOLDEN=1` to regenerate golden files and testdata `main.cli.go` files.
* Update the version in `README.md` install instructions.
* Update the `cmd/cligen` version in all testdata `go.mod` files.
* The `cmd/cligen/vX.Y.Z` tag is created by CI after the main tag is pushed.
