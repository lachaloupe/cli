package main

import (
	"context"
	"fmt"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: Search,
}

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

	// Search for pattern that do not match instead.
	//cli:alias=v
	InvertMatch bool

	// Search recursively in directories.
	//cli:alias=R
	//cli:alias=r
	Recursive bool

	// Stop after this many matches.
	//cli:alias=m
	//cli:default=10
	MaxCount uint
}

// Simple version of grep
func Search(ctx context.Context, args Args) error {
	fmt.Println(args)
	return nil
}

func main() {
	CLI.Main()
}
