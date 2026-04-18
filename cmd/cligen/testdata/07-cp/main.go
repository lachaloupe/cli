package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunCopy,
	Help:    "Copy files and directories",
}

type Args struct {
	// Preserve mode, ownership, and timestamps when possible.
	//cli:alias=a
	Archive bool

	// Overwrite destination files without prompting.
	//cli:alias=f
	Force bool

	// Prompt before overwrite.
	//cli:alias=i
	Interactive bool

	// Copy directories recursively.
	//cli:alias=R
	//cli:alias=r
	Recursive bool

	// Write a JSON manifest describing the copy plan.
	//cli:path=abs
	//cli:path=.json
	Manifest string

	// Source paths to copy.
	//cli:arg=-1
	//cli:path=rel
	//cli:path=glob
	Sources []string

	// Destination path.
	//cli:required
	//cli:arg
	//cli:path=rel
	//cli:path=mkdir
	Destination string
}

func RunCopy(ctx context.Context, args Args) error {
	fmt.Printf(
		"cp archive=%t force=%t interactive=%t recursive=%t manifest=%s sources=%s destination=%s\n",
		args.Archive,
		args.Force,
		args.Interactive,
		args.Recursive,
		args.Manifest,
		strings.Join(args.Sources, ","),
		args.Destination,
	)
	return nil
}

func main() {
	CLI.Main()
}
