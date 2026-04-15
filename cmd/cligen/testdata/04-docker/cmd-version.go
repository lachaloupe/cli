package main

import (
	"context"
	"fmt"
)

type VersionArgs struct {
	// Format the output using the given Go template.
	Format string
}

// RunVersion shows the Docker version information.
func RunVersion(ctx context.Context, args VersionArgs) error {
	fmt.Println("version")
	fmt.Println(args)
	return nil
}
