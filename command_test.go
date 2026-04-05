package cli

import (
	"net"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name: "name",
					Type: "string",
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("asdf", "--name"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--name"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("asdf", "--name", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--name", "asdf", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--name", "asdf", "--asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("--name", "asdf"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "asdf", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if cmds, err := test("--name=asdf"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "asdf", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}
}

func TestParseFlagAliases(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:    "name",
					Aliases: []string{"n", "nickname"},
					Type:    "string",
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("-n", "asdf"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "asdf", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if cmds, err := test("--nickname=hello"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}
}

func TestParseEnvironment(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")

	c := &Command{
		Args: []*Arg{
			{
				Name: "cache-dir",
				Type: "string",
				Defaults: []string{
					"$HOME/.cache/app",
				},
			},
		},
	}

	cmds, err := c.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "/tmp/test-home/.cache/app", cmds[0].Get("cache-dir").Value.(string); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestParseEnvironmentFallback(t *testing.T) {
	t.Setenv("HOME", "/tmp/test-home")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("CACHE_ROOT", "")

	c := &Command{
		Args: []*Arg{
			{
				Name: "data-dir",
				Type: "string",
				Defaults: []string{
					"$XDG_DATA_HOME/app",
					"$HOME/.local/share/app",
				},
			},
			{
				Name: "cache-dir",
				Type: "string",
				Defaults: []string{
					"$CACHE_ROOT/app",
					"$HOME/.cache/app",
				},
			},
		},
	}

	cmds, err := c.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "/tmp/test-home/.local/share/app", cmds[0].Get("data-dir").Value.(string); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}

	if want, got := "/tmp/test-home/.cache/app", cmds[0].Get("cache-dir").Value.(string); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestParseEnvironmentSkipsEmptyDefaults(t *testing.T) {
	t.Setenv("EMPTY", "")

	c := &Command{
		Args: []*Arg{
			{
				Name: "name",
				Type: "string",
				Defaults: []string{
					"$MISSING",
					"$EMPTY",
					"guest",
				},
			},
		},
	}

	cmds, err := c.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "guest", cmds[0].Get("name").Value.(string); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestPathValidateFile(t *testing.T) {
	root := t.TempDir()

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "input",
					Type:     "string",
					Labels:   []string{"path:exists", "path:file"},
					Validate: PathValidate,
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("--input", file); err != nil {
		t.Fatal(err)
	}

	if _, err := test("--input", dir); err == nil {
		t.Fatal("expected file check to fail for directory")
	}
}

func TestPathValidateDir(t *testing.T) {
	root := t.TempDir()

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "workspace",
					Type:     "string",
					Labels:   []string{"path:exists", "path:dir"},
					Validate: PathValidate,
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("--workspace", dir); err != nil {
		t.Fatal(err)
	}

	if _, err := test("--workspace", file); err == nil {
		t.Fatal("expected dir check to fail for file")
	}
}

func TestPathValidateEmptyFile(t *testing.T) {
	root := t.TempDir()

	empty := filepath.Join(root, "empty.txt")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}

	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "scratch",
					Type:     "string",
					Labels:   []string{"path:empty"},
					Validate: PathValidate,
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("--scratch", empty); err != nil {
		t.Fatal(err)
	}

	if _, err := test("--scratch", file); err == nil {
		t.Fatal("expected empty check to fail for non-empty file")
	}
}

func TestPathValidateEmptyDir(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	filled := filepath.Join(root, "filled")
	if err := os.Mkdir(filled, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(filled, "child.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "scratch",
					Type:     "string",
					Labels:   []string{"path:empty", "path:dir"},
					Validate: PathValidate,
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("--scratch", dir); err != nil {
		t.Fatal(err)
	}

	if _, err := test("--scratch", filled); err == nil {
		t.Fatal("expected empty dir check to fail for non-empty directory")
	}
}

func TestPathValidateSlice(t *testing.T) {
	root := t.TempDir()

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "paths",
					Type:     "[]string",
					Labels:   []string{"path:exists"},
					Validate: PathValidate,
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("--paths", file, "--paths", dir); err != nil {
		t.Fatal(err)
	}
}

func TestParseCustomValidate(t *testing.T) {
	seen := []string{}

	c := &Command{
		Args: []*Arg{
			{
				Name: "name",
				Type: "string",
				Validate: func(arg *Arg, s string) error {
					seen = append(seen, arg.Name+":"+s)
					if s == "bad" {
						return os.ErrInvalid
					}

					return nil
				},
			},
		},
	}

	if _, err := c.Parse([]string{"--name", "ok"}); err != nil {
		t.Fatal(err)
	}

	if want := []string{"name:ok"}; !slices.Equal(seen, want) {
		t.Fatalf("got %v; want %v", seen, want)
	}

	if _, err := c.Parse([]string{"--name", "bad"}); err == nil {
		t.Fatal(err)
	}
}

func TestParseTypes(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name: "i8",
					Type: "int8",
				},
				{
					Name: "i16",
					Type: "int16",
				},
				{
					Name: "i32",
					Type: "int32",
				},
				{
					Name: "i64",
					Type: "int64",
				},
				{
					Name: "i",
					Type: "int",
				},
				{
					Name: "f32",
					Type: "float32",
				},
				{
					Name: "f64",
					Type: "float64",
				},
				{
					Name: "c64",
					Type: "complex64",
				},
				{
					Name: "c128",
					Type: "complex128",
				},
				{
					Name: "flag",
					Type: "bool",
				},
				{
					Name: "t",
					Type: "time.Duration",
				},
				{
					Name: "mac",
					Type: "net.HardwareAddr",
				},
				{
					Name: "cidr",
					Type: "net.IPNet",
				},
				{
					Name: "endpoint",
					Type: "url.URL",
				},
				{
					Name: "from",
					Type: "mail.Address",
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test("-i8", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i8", "1e-6"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i8", "128"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-i8=42"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := int8(42), cmds[0].Get("i8").Value.(int8); got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	}

	if _, err := test("-i16", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i16", "1e-6"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i16", "32768"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-i16", "42"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := int16(42), cmds[0].Get("i16").Value.(int16); got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	}

	if _, err := test("-i32", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i32", "1e-6"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i32", "2147483648"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-i32=42"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := int32(42), cmds[0].Get("i32").Value.(int32); got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	}

	if _, err := test("-i64", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i64", "1e-6"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i64", "9223372036854775808"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-i64=42"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := int64(42), cmds[0].Get("i64").Value.(int64); got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	}

	if _, err := test("-i", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i", "1e-6"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("-i", "9223372036854775808"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-i=42"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := 42, cmds[0].Get("i").Value.(int); got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	}

	if _, err := test("-f32", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-f32=1e6"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := float32(1e6), cmds[0].Get("f32").Value.(float32); got != want {
			t.Fatalf("got %f; want %f", got, want)
		}
	}

	if _, err := test("-f64", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-f64=1e6"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := 1e6, cmds[0].Get("f64").Value.(float64); got != want {
			t.Fatalf("got %f; want %f", got, want)
		}
	}

	if _, err := test("-c64", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-c64", "1+2i"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := complex64(1+2i), cmds[0].Get("c64").Value.(complex64); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if _, err := test("-c128", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-c128", "1+2i"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := complex128(1+2i), cmds[0].Get("c128").Value.(complex128); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if cmds, err := test("-flag"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := true, cmds[0].Get("flag").Value.(bool); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if cmds, err := test("-flag=false"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := false, cmds[0].Get("flag").Value.(bool); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if cmds, err := test("-flag=0"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := false, cmds[0].Get("flag").Value.(bool); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if _, err := test("-flag", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-t", "1s"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := time.Second, cmds[0].Get("t").Value.(time.Duration); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if _, err := test("-t", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-mac", "01:23:45:67:89:ab"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "01:23:45:67:89:ab", cmds[0].Get("mac").Value.(net.HardwareAddr).String(); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if _, err := test("-mac", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-cidr", "192.0.2.1/24"); err != nil {
		t.Fatal(err)
	} else {
		got := cmds[0].Get("cidr").Value.(net.IPNet)

		if want, got := "192.0.2.0/24", (&got).String(); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if _, err := test("-cidr", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-endpoint", "https://example.com/path?q=1"); err != nil {
		t.Fatal(err)
	} else {
		got := cmds[0].Get("endpoint").Value.(url.URL)

		if want, got := "https://example.com/path?q=1", (&got).String(); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if _, err := test("-endpoint", "://bad"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("-from", "Alice <alice@example.com>"); err != nil {
		t.Fatal(err)
	} else {
		got := cmds[0].Get("from").Value.(mail.Address)

		if want, got := "\"Alice\" <alice@example.com>", (&got).String(); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if _, err := test("-from", "not-an-address"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCommands(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name: "name",
					Type: "string",
				},
			},
			Commands: []*Command{
				{
					Name: "a",
					Args: []*Arg{
						{
							Name: "name",
							Type: "string",
						},
					},
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("--name", "hello", "a", "--name", "world"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}

		if want, got := "world", cmds[1].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}
}

func TestParseCommandAliases(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Name: "test",
			Commands: []*Command{
				{
					Name:    "login",
					Aliases: []string{"signin", "in"},
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("signin"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "login", cmds[1].Name; got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if cmds, err := test("in"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "login", cmds[1].Name; got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}
}

func TestParsePositionals(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:       "name",
					Type:       "string",
					Positional: 1,
				},
			},
			Commands: []*Command{
				{
					Name: "a",
					Args: []*Arg{
						{
							Name:       "value",
							Type:       "int",
							Positional: 1,
						},
					},
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("hello"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if cmds, err := test("hello", "a"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}

		if got := cmds[1].Get("value").Value; got != nil {
			t.Fatalf("got %v; want nil", got)
		}
	}

	if _, err := test("hello", "a", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("hello", "a", "1", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("hello", "a", "--", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("hello", "a", "--", "-1", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("hello", "a", "1"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("name").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}

		if want, got := 1, cmds[1].Get("value").Value.(int); got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

func TestParsePositionalsWithTail(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:       "pair",
					Type:       "[]string",
					Positional: 2,
				},
				{
					Name:       "tail",
					Type:       "[]int",
					Positional: -1,
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("foo", "bar", "1", "2", "3"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := []string{"foo", "bar"}, cmds[0].Get("pair").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		if want, got := []int{1, 2, 3}, cmds[0].Get("tail").Value.([]int); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if _, err := test("foo", "bar", "1", "2", "asdf"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("foo", "bar", "--", "-1", "-2", "-3"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := []string{"foo", "bar"}, cmds[0].Get("pair").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		if want, got := []int{-1, -2, -3}, cmds[0].Get("tail").Value.([]int); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

func TestParsePositionalsWithHead(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:       "head",
					Type:       "[]string",
					Positional: -1,
				},
				{
					Name:       "pair",
					Type:       "[]string",
					Positional: 2,
				},
				{
					Name:       "last",
					Type:       "string",
					Positional: 1,
				},
			},
		}

		return c.Parse(args)
	}

	if cmds, err := test("a", "b", "c"); err != nil {
		t.Fatal(err)
	} else {
		if got := cmds[0].Get("head").Value; got != nil {
			t.Fatalf("got %v; want nil", got)
		}

		if want, got := []string{"a", "b"}, cmds[0].Get("pair").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		if want, got := "c", cmds[0].Get("last").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}

	if cmds, err := test("x", "y", "a", "b", "c"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := []string{"x", "y"}, cmds[0].Get("head").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		if want, got := []string{"a", "b"}, cmds[0].Get("pair").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		if want, got := "c", cmds[0].Get("last").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}
	}
}

func TestParseRequired(t *testing.T) {
	test := func(args ...string) ([]*Command, error) {
		c := &Command{
			Args: []*Arg{
				{
					Name:     "flag",
					Type:     "string",
					Required: true,
				},
				{
					Name:       "strings",
					Type:       "[]string",
					Positional: 2,
					Required:   true,
				},
			},
			Commands: []*Command{
				{
					Name: "a",
					Args: []*Arg{
						{
							Name:       "items",
							Type:       "[]string",
							Positional: -1,
							Required:   true,
						},
					},
				},
			},
		}

		return c.Parse(args)
	}

	if _, err := test(); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--flag", "hello"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("a"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--flag"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := test("--flag", "hello", "world"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("--flag", "hello", "foo", "bar"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("flag").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}

		if want, got := []string{"foo", "bar"}, cmds[0].Get("strings").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	}

	if _, err := test("--flag", "hello", "foo", "bar", "a"); err == nil {
		t.Fatal("expected error")
	}

	if cmds, err := test("--flag", "hello", "foo", "bar", "a", "coco"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := "hello", cmds[0].Get("flag").Value.(string); got != want {
			t.Fatalf("got %s; want %s", got, want)
		}

		if want, got := []string{"coco"}, cmds[1].Get("items").Value.([]string); !slices.Equal(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

func TestHelp(t *testing.T) {
	test := func(args ...string) string {
		c := &Command{
			Name: "test",
			Help: "this is a test",
			Args: []*Arg{
				{
					Name: "name",
					Type: "string",
					Help: "your name",
				},
			},
			Commands: []*Command{
				{
					Name: "a",
					Help: "some command",
					Args: []*Arg{
						{
							Name: "flag",
							Type: "string",
							Help: "some flag",
						},
					},
				},
			},
		}

		cmds, err := c.Parse(args)
		if err != ErrHelp {
			t.Fatal("expected help")
		}

		return Help(cmds)
	}

	helps := []string{
		`Usage: test

this is a test

Options:
  --name           your name (string)

Subcommands:
  a                some command
`,
		`Usage: test a

some command

Options:
  --flag           some flag (string)

Options (global):
  --name           your name (string)
`,
	}

	if got, want := test("--help"), helps[0]; got != want {
		t.Fatalf("got:\n%s", got)
	}

	if got, want := test("a", "--help"), helps[1]; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
