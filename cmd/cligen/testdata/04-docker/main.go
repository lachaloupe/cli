package main

import (
	"context"
	"fmt"
	"strings"

	"example.com/testcli/container"
	"example.com/testcli/image"
	"example.com/testcli/network"
	"example.com/testcli/volume"
	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunDocker,
	Help:    "A self-sufficient runtime for containers",
	Commands: []*cli.Command{
		{
			Name: "image",
			Help: "Manage images",
			Commands: []*cli.Command{
				{
					Name:    "ls",
					Handler: image.RunLs,
				},
				{
					Name:    "pull",
					Handler: image.RunPull,
				},
				{
					Name:    "tag",
					Handler: image.RunTag,
				},
				{
					Name:    "rm",
					Handler: image.RunRm,
				},
			},
		},
		{
			Name: "container",
			Help: "Manage containers",
			Commands: []*cli.Command{
				{
					Name:    "ls",
					Handler: container.RunLs,
				},
				{
					Name:    "run",
					Handler: container.RunRun,
				},
				{
					Name:    "logs",
					Handler: container.RunLogs,
				},
				{
					Name:    "rm",
					Handler: container.RunRm,
				},
			},
		},
		{
			Name: "network",
			Help: "Manage networks",
			Commands: []*cli.Command{
				{
					Name:    "ls",
					Handler: network.RunLs,
				},
				{
					Name:    "create",
					Handler: network.RunCreate,
				},
				{
					Name:    "rm",
					Handler: network.RunRm,
				},
			},
		},
		{
			Name: "volume",
			Help: "Manage volumes",
			Commands: []*cli.Command{
				{
					Name:    "ls",
					Handler: volume.RunLs,
				},
				{
					Name:    "create",
					Handler: volume.RunCreate,
				},
				{
					Name:    "rm",
					Handler: volume.RunRm,
				},
			},
		},
		{
			Name: "system",
			Help: "Manage Docker",
			Commands: []*cli.Command{
				{
					Name:    "prune",
					Handler: RunSystemPrune,
				},
			},
		},
		{
			Name:    "version",
			Handler: RunVersion,
		},
	},
}

type DockerArgs struct {
	// Set the API version to use.
	ApiVersion string

	// Location of client configuration files.
	//cli:path=clean
	Config string

	// Enable debug mode.
	//cli:alias=D
	Debug bool

	// Daemon socket to connect to.
	//cli:alias=H
	//cli:default=unix:///var/run/docker.sock
	Host []string

	// Use TLS.
	Tls bool

	// Trust certs signed only by this CA.
	//cli:path=file
	//cli:path=clean
	Tlscacert string
}

// RunDocker runs the root Docker command.
func RunDocker(ctx context.Context, args DockerArgs) error {
	fmt.Printf(
		"docker api-version=%s config=%s debug=%t host=%s tls=%t tlscacert=%s\n",
		args.ApiVersion,
		args.Config,
		args.Debug,
		strings.Join(args.Host, ","),
		args.Tls,
		args.Tlscacert,
	)
	return nil
}

func main() {
	CLI.Main()
}
