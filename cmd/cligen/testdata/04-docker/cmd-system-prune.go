package main

import (
	"context"
	"fmt"
)

type SystemPruneArgs struct {
	// Remove all unused images, not just dangling ones.
	//cli:alias=a
	All bool

	// Provide filter values.
	Filter []string

	// Do not prompt for confirmation.
	//cli:alias=f
	Force bool

	// Prune anonymous volumes.
	Volumes bool
}

// RunSystemPrune removes unused data.
func RunSystemPrune(ctx context.Context, args SystemPruneArgs) error {
	fmt.Println("system prune")
	fmt.Println(args)
	return nil
}
