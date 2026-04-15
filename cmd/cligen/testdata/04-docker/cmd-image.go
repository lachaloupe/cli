package main

import (
	"context"
	"fmt"
)

type ImageLsArgs struct {
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
func RunImageLs(ctx context.Context, args ImageLsArgs) error {
	fmt.Println("image ls")
	fmt.Println(args)
	return nil
}

type ImagePullArgs struct {
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
func RunImagePull(ctx context.Context, args ImagePullArgs) error {
	fmt.Println("image pull")
	fmt.Println(args)
	return nil
}

type ImageRmArgs struct {
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
func RunImageRm(ctx context.Context, args ImageRmArgs) error {
	fmt.Println("image rm")
	fmt.Println(args)
	return nil
}
