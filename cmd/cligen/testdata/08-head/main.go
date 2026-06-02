package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunHead,
	New:     NewArgs,
	Help:    "Output the first part of files",
}

func NewArgs() Args {
	return Args{
		Files: []io.Reader{os.Stdin},
	}
}

type Args struct {
	// Print the first NUM bytes of each file.
	//cli:alias=c
	Bytes uint

	// Print the first NUM lines instead of the first 10.
	//cli:alias=n
	//cli:default=10
	Lines uint

	// Never print headers with file names.
	//cli:alias=q
	Quiet bool

	// Always print headers with file names.
	//cli:alias=v
	Verbose bool

	// Files to read. Reads standard input when omitted.
	//cli:arg
	Files []io.Reader
}

// RunHead prints the first part of files.
func RunHead(ctx context.Context, args Args) error {
	w := cli.Stdout(ctx)

	showHeaders := args.Verbose || (!args.Quiet && len(args.Files) > 1)

	for i, file := range args.Files {
		if showHeaders {
			if i > 0 {
				fmt.Fprintln(w)
			}

			name := "-"
			if f, ok := file.(*os.File); ok && f != os.Stdin {
				name = f.Name()
			}

			fmt.Fprintf(w, "==> %s <==\n", name)
		}

		if args.Bytes > 0 {
			if _, err := io.CopyN(w, file, int64(args.Bytes)); err != nil && !errors.Is(err, io.EOF) {
				return err
			}

			continue
		}

		scanner := bufio.NewScanner(file)

		for n := uint(0); n < args.Lines && scanner.Scan(); n++ {
			fmt.Fprintln(w, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	CLI.Main()
}
