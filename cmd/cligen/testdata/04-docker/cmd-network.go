package main

import (
	"context"
	"fmt"
	"net"
)

type NetworkLsArgs struct {
	// Provide filter values.
	//cli:alias=f
	Filter []string

	// Pretty-print networks using a Go template.
	Format string

	// Only display network IDs.
	//cli:alias=q
	Quiet bool
}

// RunNetworkLs lists networks.
func RunNetworkLs(ctx context.Context, args NetworkLsArgs) error {
	fmt.Println("network ls")
	fmt.Println(args)
	return nil
}

type NetworkCreateArgs struct {
	// Driver to manage the network.
	//cli:default=bridge
	Driver string

	// Set metadata on the network.
	//cli:alias=l
	Label []string

	// Restrict external access to the network.
	Internal bool

	// Assign a preferred MAC address to the network gateway.
	GatewayMAC net.HardwareAddr

	// Allocate addresses from this subnet.
	Subnet net.IPNet

	// Restrict dynamic allocation to this subrange.
	IPRange net.IPNet

	// Network name.
	//cli:required
	//cli:arg
	Name string
}

// RunNetworkCreate creates a network.
func RunNetworkCreate(ctx context.Context, args NetworkCreateArgs) error {
	fmt.Printf(
		"network-create driver=%s internal=%t gateway-mac=%s subnet=%s ip-range=%s labels=%s name=%s\n",
		args.Driver,
		args.Internal,
		args.GatewayMAC.String(),
		args.Subnet.String(),
		args.IPRange.String(),
		fmt.Sprint(args.Label),
		args.Name,
	)
	return nil
}

type NetworkRmArgs struct {
	// Network names or IDs.
	//cli:required
	//cli:arg
	Networks []string
}

// RunNetworkRm removes networks.
func RunNetworkRm(ctx context.Context, args NetworkRmArgs) error {
	fmt.Println("network rm")
	fmt.Println(args)
	return nil
}
