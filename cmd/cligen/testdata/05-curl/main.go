package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunCurl,
	Help:    "Transfer data from or to a server",
}

type Args struct {
	// Request method to use.
	//cli:alias=X
	//cli:default=GET
	Method string

	// Pass custom header fields to the server.
	//cli:alias=H
	Header []string

	// Send data in the request body.
	//cli:alias=d
	Data string

	// Follow redirects.
	//cli:alias=L
	Location bool

	// Maximum time allowed for connection setup.
	//cli:default=5s
	ConnectTimeout time.Duration

	// Retry this many times on transient failures.
	Retry uint

	// Write output to this file.
	//cli:alias=o
	//cli:path=creatable
	//cli:path=clean
	Output string

	// Trust certificates signed only by this CA bundle.
	//cli:path=exists
	//cli:path=file
	//cli:path=readable
	//cli:path=clean
	Cacert string

	// Use the given proxy for the request.
	Proxy url.URL

	// Send this user agent to the server.
	//cli:alias=A
	//cli:default=curl/8.0
	UserAgent string

	// URL to fetch.
	//cli:required
	//cli:arg
	URL url.URL
}

// RunCurl runs a single HTTP request.
func RunCurl(ctx context.Context, args Args) error {
	fmt.Printf(
		"curl method=%s headers=%s data=%s location=%t connect-timeout=%s retry=%d output=%s cacert=%s proxy=%s user-agent=%s url=%s\n",
		args.Method,
		strings.Join(args.Header, ","),
		args.Data,
		args.Location,
		args.ConnectTimeout,
		args.Retry,
		args.Output,
		args.Cacert,
		args.Proxy.String(),
		args.UserAgent,
		args.URL.String(),
	)
	return nil
}

func main() {
	CLI.Main()
}
