module example.com/testcli

go 1.26.3

replace github.com/lachaloupe/cli => ../../../..

replace github.com/lachaloupe/cli/cmd/cligen => ../..

require github.com/lachaloupe/cli v0.1.5

require (
	github.com/lachaloupe/cli/cmd/cligen v0.1.5 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

tool github.com/lachaloupe/cli/cmd/cligen
