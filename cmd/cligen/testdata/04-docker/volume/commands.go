package volume

import (
	"context"
	"fmt"
)

type LsArgs struct {
	// Provide filter values.
	//cli:alias=f
	Filter []string

	// Pretty-print volumes using a Go template.
	Format string

	// Only display volume names.
	//cli:alias=q
	Quiet bool
}

// RunVolumeLs lists volumes.
func RunLs(ctx context.Context, args LsArgs) error {
	fmt.Println("volume ls")
	fmt.Println(args)
	return nil
}

type CreateArgs struct {
	// Driver to manage the volume.
	//cli:default=local
	Driver string

	// Set metadata on the volume.
	//cli:alias=l
	Label []string

	// Volume name.
	//cli:arg
	Name string
}

// RunVolumeCreate creates a volume.
func RunCreate(ctx context.Context, args CreateArgs) error {
	fmt.Println("volume create")
	fmt.Println(args)
	return nil
}

type RmArgs struct {
	// Force removal of the volume.
	//cli:alias=f
	Force bool

	// Volume names.
	//cli:required
	//cli:arg
	Volumes []string
}

// RunVolumeRm removes volumes.
func RunRm(ctx context.Context, args RmArgs) error {
	fmt.Println("volume rm")
	fmt.Println(args)
	return nil
}
