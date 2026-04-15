package main

import (
	"context"
	"fmt"
)

type VolumeLsArgs struct {
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
func RunVolumeLs(ctx context.Context, args VolumeLsArgs) error {
	fmt.Println("volume ls")
	fmt.Println(args)
	return nil
}

type VolumeCreateArgs struct {
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
func RunVolumeCreate(ctx context.Context, args VolumeCreateArgs) error {
	fmt.Println("volume create")
	fmt.Println(args)
	return nil
}

type VolumeRmArgs struct {
	// Force removal of the volume.
	//cli:alias=f
	Force bool

	// Volume names.
	//cli:required
	//cli:arg
	Volumes []string
}

// RunVolumeRm removes volumes.
func RunVolumeRm(ctx context.Context, args VolumeRmArgs) error {
	fmt.Println("volume rm")
	fmt.Println(args)
	return nil
}
