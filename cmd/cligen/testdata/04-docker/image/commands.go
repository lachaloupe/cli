package image

import (
	"context"
	"fmt"
)

type LsArgs struct {
	// Show all images.
	//cli:alias=a
	All bool

	// Show digests.
	Digests bool

	// Pretty-print images using a Go template.
	Format string

	// Don't truncate output.
	NoTrunc bool

	// Only show image IDs.
	//cli:alias=q
	Quiet bool

	// Provide filter values.
	//cli:alias=f
	Filter []string
}

// RunImageLs lists images.
func RunLs(ctx context.Context, args LsArgs) error {
	fmt.Println("image ls")
	fmt.Println(args)
	return nil
}

type PullArgs struct {
	// Download all tagged images in the repository.
	//cli:alias=a
	AllTags bool

	// Set platform if server is multi-platform capable.
	Platform string

	// Suppress verbose output.
	//cli:alias=q
	Quiet bool

	// Image name to pull.
	//cli:required
	//cli:arg
	Image string
}

// RunImagePull pulls an image.
func RunPull(ctx context.Context, args PullArgs) error {
	fmt.Println("image pull")
	fmt.Println(args)
	return nil
}

type TagArgs struct {
	// Source and target image references.
	//cli:arg=2
	References []string
}

// RunImageTag tags an image into a repository.
func RunTag(ctx context.Context, args TagArgs) error {
	fmt.Printf("image tag source=%s target=%s\n", args.References[0], args.References[1])
	return nil
}

type RmArgs struct {
	// Force removal of the image.
	//cli:alias=f
	Force bool

	// Do not delete untagged parents.
	NoPrune bool

	// Image references to remove.
	//cli:required
	//cli:arg
	Images []string
}

// RunImageRm removes images.
func RunRm(ctx context.Context, args RmArgs) error {
	fmt.Println("image rm")
	fmt.Println(args)
	return nil
}
