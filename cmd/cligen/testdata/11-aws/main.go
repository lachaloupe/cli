package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen -provider aws

var CLI = cli.Command{
	Handler: RunCat,
	Help:    "Print the contents of HTTP or S3 objects",
}

type Args struct {
	// Files to print.
	//cli:required
	//cli:arg
	Files []io.Reader
}

func RunCat(ctx context.Context, args Args) error {
	parts := make([]string, 0, len(args.Files))
	for _, file := range args.Files {
		body, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		parts = append(parts, strconv.Quote(strings.TrimSpace(string(body))))
	}

	fmt.Printf("cat files=%d contents=%s\n", len(parts), strings.Join(parts, ","))
	return nil
}

func main() {
	CLI.Main()
}
