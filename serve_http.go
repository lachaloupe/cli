package cli

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

const serveHTTPShutdownTimeout = 5 * time.Second

// ServeHTTPArgs configures the built-in HTTP server handler.
type ServeHTTPArgs struct {
	// Address to listen on.
	//cli:default=127.0.0.1:8080
	Addr string
}

// MakeServeHTTP returns a handler that serves the provided HTTP handler until
// the command context is canceled.
func MakeServeHTTP(handler http.Handler) func(context.Context, ServeHTTPArgs) error {
	return func(ctx context.Context, args ServeHTTPArgs) error {
		h := handler
		if h == nil {
			h = http.DefaultServeMux
		}

		ln, err := net.Listen("tcp", args.Addr)
		if err != nil {
			return err
		}

		srv := &http.Server{
			Addr:    ln.Addr().String(),
			Handler: h,
			BaseContext: func(net.Listener) context.Context {
				return ctx
			},
		}

		done := make(chan error, 1)
		go func() {
			done <- srv.Serve(ln)
		}()

		select {
		case err := <-done:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}

			return err
		case <-ctx.Done():
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), serveHTTPShutdownTimeout)
		defer cancel()

		err = srv.Shutdown(shutdownCtx)

		serveErr := <-done
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}

		return errors.Join(serveErr, err)
	}
}
