package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (c *Command) lookupValue(ctx context.Context, arg *Arg, s string) (string, error) {
	if c.Lookup != nil {
		if value, ok, err := c.Lookup(ctx, arg, s); err != nil {
			return "", err
		} else if ok {
			return value, nil
		}
	}

	if strings.HasPrefix(s, "@@") {
		return s[1:], nil
	}

	if !strings.HasPrefix(s, "@") {
		return s, nil
	}

	p := strings.TrimPrefix(s, "@")
	if p == "" {
		return "", fmt.Errorf("%s: empty file reference", arg.Name)
	}

	body, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("%s: read %q: %w", arg.Name, p, err)
	}

	return string(body), nil
}

func (c *Command) openReader(ctx context.Context, arg *Arg, s string) (io.Reader, error) {
	if c.Open != nil {
		if reader, ok, err := c.Open(ctx, arg, s); err != nil {
			return nil, err
		} else if ok {
			if closer, ok := reader.(io.Closer); ok {
				c.Cleanups = append(c.Cleanups, closer.Close)
			}

			return reader, nil
		}
	}

	if s == "-" {
		return os.Stdin, nil
	}

	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		switch u.Scheme {
		case "http", "https":
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, nil)
			if err != nil {
				return nil, fmt.Errorf("%s: build GET %q: %w", arg.Name, s, err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("%s: GET %q: %w", arg.Name, s, err)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close()
				return nil, fmt.Errorf("%s: GET %q: unexpected status %s", arg.Name, s, resp.Status)
			}

			c.Cleanups = append(c.Cleanups, resp.Body.Close)
			return resp.Body, nil
		case "file":
			path := u.Path
			if path == "" {
				return nil, fmt.Errorf("%s: empty file URI %q", arg.Name, s)
			}
			if u.Host != "" && u.Host != "localhost" {
				return nil, fmt.Errorf("%s: unsupported file URI host %q", arg.Name, u.Host)
			}

			file, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("%s: open %q: %w", arg.Name, path, err)
			}

			c.Cleanups = append(c.Cleanups, file.Close)
			return file, nil
		}
	}

	file, err := os.Open(s)
	if err != nil {
		return nil, fmt.Errorf("%s: open %q: %w", arg.Name, s, err)
	}

	c.Cleanups = append(c.Cleanups, file.Close)
	return file, nil
}
