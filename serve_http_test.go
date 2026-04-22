package cli

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestMakeServeHTTPServesAndShutsDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- MakeServeHTTP(mux)(ctx, ServeHTTPArgs{Addr: addr})
	}()

	client := &http.Client{Timeout: time.Second}

	var resp *http.Response
	for range 50 {
		resp, err = client.Get("http://" + addr + "/healthz")
		if err == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}

	if want, got := http.StatusNoContent, resp.StatusCode; got != want {
		t.Fatalf("got status %d; want %d", got, want)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	cancel()

	if err := <-done; err != nil {
		t.Fatalf("MakeServeHTTP: %v", err)
	}
}
