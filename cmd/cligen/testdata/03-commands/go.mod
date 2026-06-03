module example.com/testcli

go 1.26.3

replace github.com/lachaloupe/cli => ../../../..

replace github.com/lachaloupe/cli/cmd/cligen => ../..

require github.com/lachaloupe/cli v0.1.3

require (
	github.com/lachaloupe/cli/cmd/cligen v0.1.3 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
)

tool github.com/lachaloupe/cli/cmd/cligen
