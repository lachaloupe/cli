package container

import (
	"context"
	"fmt"
	"strings"
)

type LsArgs struct {
	// Show all containers.
	//cli:alias=a
	All bool

	// Provide filter values.
	//cli:alias=f
	Filter []string

	// Pretty-print containers using a Go template.
	Format string

	// Show n last created containers.
	//cli:alias=n
	Last int

	// Only display container IDs.
	//cli:alias=q
	Quiet bool

	// Display total file sizes.
	Size bool
}

// RunContainerLs lists containers.
func RunLs(ctx context.Context, args LsArgs) error {
	fmt.Println("container ls")
	fmt.Println(args)
	return nil
}

type RunArgs struct {
	// Run container in background and print container ID.
	//cli:alias=d
	Detach bool

	// Set environment variables.
	//cli:alias=e
	Env []string

	// Read in a file of environment variables.
	//cli:path=exists
	//cli:path=file
	EnvFile []string

	// Keep STDIN open even if not attached.
	//cli:alias=i
	Interactive bool

	// Assign a name to the container.
	Name string

	// Publish a container's port to the host.
	//cli:alias=p
	Publish []string

	// Automatically remove the container when it exits.
	Rm bool

	// Pull image before running.
	//cli:enum=always
	//cli:enum=missing
	//cli:enum=never
	//cli:default=missing
	Pull string

	// Allocate a pseudo-TTY.
	//cli:alias=t
	Tty bool

	// Bind mount a volume.
	//cli:alias=v
	Volume []string

	// Working directory inside the container.
	//cli:alias=w
	Workdir string

	// Image to run.
	//cli:required
	//cli:arg=1
	Image string

	// Command and arguments to execute.
	//cli:arg=-1
	Command []string
}

// RunContainerRun runs a command in a new container.
func RunRun(ctx context.Context, args RunArgs) error {
	fmt.Printf(
		"container-run detach=%t env=%s env-file=%s interactive=%t name=%s publish=%s rm=%t pull=%s tty=%t volume=%s workdir=%s image=%s command=%s\n",
		args.Detach,
		strings.Join(args.Env, ","),
		strings.Join(args.EnvFile, ","),
		args.Interactive,
		args.Name,
		strings.Join(args.Publish, ","),
		args.Rm,
		args.Pull,
		args.Tty,
		strings.Join(args.Volume, ","),
		args.Workdir,
		args.Image,
		strings.Join(args.Command, " "),
	)
	return nil
}

type LogsArgs struct {
	// Follow log output.
	//cli:alias=f
	Follow bool

	// Show logs since timestamp or relative duration.
	Since string

	// Number of lines to show from the end of the logs.
	//cli:default=all
	Tail string

	// Show timestamps.
	//cli:alias=t
	Timestamps bool

	// Container name or ID.
	//cli:required
	//cli:arg
	Container string
}

// RunContainerLogs fetches container logs.
func RunLogs(ctx context.Context, args LogsArgs) error {
	fmt.Println("container logs")
	fmt.Println(args)
	return nil
}

type RmArgs struct {
	// Force removal of a running container.
	//cli:alias=f
	Force bool

	// Remove anonymous volumes associated with the container.
	//cli:alias=v
	Volumes bool

	// Container names or IDs.
	//cli:required
	//cli:arg
	Containers []string
}

// RunContainerRm removes containers.
func RunRm(ctx context.Context, args RmArgs) error {
	fmt.Println("container rm")
	fmt.Println(args)
	return nil
}
