module example.com/testcli

go 1.26.2

replace github.com/lachaloupe/cli => ../../../..

replace github.com/lachaloupe/cli/cmd/cligen => ../..

require github.com/lachaloupe/cli v0.1.0

require (
	github.com/lachaloupe/cli/cmd/cligen v0.0.0-20260420021711-73d7228cbd26 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
)

tool github.com/lachaloupe/cli/cmd/cligen
