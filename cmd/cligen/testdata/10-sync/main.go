package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunSync,
	Help:    "Synchronize a working tree into a release directory",
}

type Args struct {
	// Root directory containing the files to sync.
	//cli:path=exists
	//cli:path=dir
	Worktree string

	// Number of concurrent copy workers.
	Workers int

	// Retry failed copies this many times.
	Retries uint8

	// Lower values run helper work sooner.
	Priority int8

	// Empty scratch directory used while staging the release.
	//cli:path=exists
	//cli:path=dir
	//cli:path=empty
	Scratch string

	// Existing state file to update while the sync runs.
	//cli:path=exists
	State string

	// Helper program used to post-process copied files.
	//cli:path=exec
	Helper string

	// Lock file path reserved for the next sync run.
	//cli:path=not-exists
	Lock string

	// Symlink pointing at the current release.
	//cli:path=symlink
	Current string

	// Source patterns to include in the sync.
	//cli:arg=-1
	Sources []string

	// Destination directory name for the staged release.
	//cli:required
	//cli:arg
	Destination string
}

func RunSync(ctx context.Context, args Args) error {
	fmt.Printf(
		"sync worktree=%s workers=%d retries=%d priority=%d scratch=%s state=%s helper=%s lock=%s current=%s sources=%s destination=%s\n",
		args.Worktree,
		args.Workers,
		args.Retries,
		args.Priority,
		args.Scratch,
		args.State,
		args.Helper,
		args.Lock,
		args.Current,
		strings.Join(args.Sources, ","),
		args.Destination,
	)
	return nil
}

func main() {
	CLI.Main()
}
