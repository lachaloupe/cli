package main

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/lachaloupe/cli"
)

type CleanupMode string

const (
	CleanupStrip      CleanupMode = "strip"
	CleanupWhitespace CleanupMode = "whitespace"
	CleanupVerbatim   CleanupMode = "verbatim"
)

func (m CleanupMode) String() string {
	return string(m)
}

func (CleanupMode) Strings() []string {
	return []string{
		string(CleanupStrip),
		string(CleanupWhitespace),
		string(CleanupVerbatim),
	}
}

func (m *CleanupMode) UnmarshalText(text []byte) error {
	switch s := string(text); s {
	case string(CleanupStrip):
		*m = CleanupStrip
	case string(CleanupWhitespace):
		*m = CleanupWhitespace
	case string(CleanupVerbatim):
		*m = CleanupVerbatim
	default:
		return fmt.Errorf("unknown cleanup mode %q", s)
	}

	return nil
}

//go:generate go tool cligen

var CLI = cli.Command{
	Help: "Fast, scalable, distributed revision control system",
	Commands: []*cli.Command{
		{
			Name:    "commit",
			Handler: RunCommit,
		},
		{
			Name: "remote",
			Help: "Manage set of tracked repositories",
			Commands: []*cli.Command{
				{
					Name:    "add",
					Handler: RunRemoteAdd,
				},
			},
		},
	},
}

type CommitArgs struct {
	// Automatically stage tracked files that have been modified and deleted.
	//cli:alias=a
	All bool

	// Replace the tip of the current branch by creating a new commit.
	Amend bool

	// Determine how the commit message is cleaned up.
	//cli:enum
	//cli:default=strip
	Cleanup CleanupMode

	// Override the commit author.
	Author mail.Address

	// Use the given message as the commit message.
	//cli:alias=m
	//cli:default=$GIT_MESSAGE
	//cli:default=@COMMIT_EDITMSG
	Message string

	// Add a Signed-off-by trailer.
	//cli:alias=s
	Signoff bool

	// Use this template file when preparing the message.
	//cli:path=exists
	//cli:path=file
	//cli:path=readable
	//cli:path=clean
	Template string

	// Update the index before committing these paths.
	//cli:arg
	Paths []string
}

// RunCommit creates a new commit.
func RunCommit(ctx context.Context, args CommitArgs) error {
	author := ""
	if args.Author.Address != "" || args.Author.Name != "" {
		author = args.Author.String()
	}

	fmt.Printf(
		"commit all=%t amend=%t cleanup=%s author=%s message=%s signoff=%t template=%s paths=%s\n",
		args.All,
		args.Amend,
		args.Cleanup,
		author,
		strings.TrimSpace(args.Message),
		args.Signoff,
		args.Template,
		strings.Join(args.Paths, ","),
	)
	return nil
}

type RemoteAddArgs struct {
	// Fetch the remote branches immediately after adding the remote.
	//cli:alias=f
	Fetch bool

	// Name of the new remote.
	//cli:required
	//cli:arg
	Name string

	// URL of the remote repository.
	//cli:required
	//cli:arg
	URL url.URL
}

// RunRemoteAdd adds a new remote.
func RunRemoteAdd(ctx context.Context, args RemoteAddArgs) error {
	fmt.Printf("remote-add fetch=%t name=%s url=%s\n", args.Fetch, args.Name, args.URL.String())
	return nil
}

func main() {
	CLI.Main()
}
