package cli

import (
	"context"
	"errors"
	"testing"
)

func TestParseErrorDetails(t *testing.T) {
	c := &Command{
		Args: []*Arg{
			{Name: "name", Type: "string", Required: true},
		},
	}

	_, err := c.Parse(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrMissingRequired) {
		t.Fatalf("got %v; want %v", err, ErrMissingRequired)
	}

	var pe *ParseError

	if !errors.As(err, &pe) {
		t.Fatalf("got %T; want *ParseError", err)
	}

	if want, got := "name", pe.Name; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestParseTypeErrorsWrapArgContext(t *testing.T) {
	c := &Command{
		Args: []*Arg{
			{Name: "count", Type: "int"},
		},
	}

	_, err := c.Parse(context.Background(), []string{"--count", "nope"})
	if err == nil {
		t.Fatal("expected error")
	}

	var ae *ArgError

	if !errors.As(err, &ae) {
		t.Fatalf("got %T; want *ArgError", err)
	}

	if want, got := "count", ae.Arg; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}

	if want, got := "nope", ae.Value; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}
