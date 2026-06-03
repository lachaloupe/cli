package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func (c *Command) lookupValue(ctx context.Context, arg *Arg, s string) (string, error) {
	if c.Lookup != nil {
		value, ok, err := c.Lookup(ctx, arg, s)
		if err != nil {
			return "", err
		}

		if ok {
			return value, nil
		}
	}

	p, ok := strings.CutPrefix(s, "@")
	if !ok {
		return s, nil
	}

	if strings.HasPrefix(p, "@") {
		return p, nil
	}

	if p == "" {
		return "", fmt.Errorf("empty file reference")
	}

	body, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *Command) openReader(ctx context.Context, arg *Arg, s string) (io.Reader, error) {
	if c.Open != nil {
		r, ok, err := c.Open(ctx, arg, s)
		if err != nil {
			return nil, err
		}

		if ok {
			if closer, ok := r.(io.Closer); ok {
				c.Cleanups = append(c.Cleanups, closer.Close)
			}

			return r, nil
		}
	}

	if s == "-" {
		return Stdin(ctx), nil
	}

	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		switch u.Scheme {
		case "http", "https":
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, nil)
			if err != nil {
				return nil, fmt.Errorf("build GET %q: %w", s, err)
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("GET %q: %w", s, err)
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close()
				return nil, fmt.Errorf("GET %q: unexpected status %s", s, resp.Status)
			}

			c.Cleanups = append(c.Cleanups, resp.Body.Close)
			return resp.Body, nil
		case "file":
			path := u.Path

			if path == "" {
				return nil, fmt.Errorf("empty file URI %q", s)
			}
			if u.Host != "" && u.Host != "localhost" {
				return nil, fmt.Errorf("unsupported file URI host %q", u.Host)
			}

			file, err := os.Open(path)
			if err != nil {
				return nil, err
			}

			c.Cleanups = append(c.Cleanups, file.Close)
			return file, nil
		}
	}

	file, err := os.Open(s)
	if err != nil {
		return nil, err
	}

	c.Cleanups = append(c.Cleanups, file.Close)
	return file, nil
}
