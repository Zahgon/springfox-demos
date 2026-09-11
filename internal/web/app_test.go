package web

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestRunServesAndShutsDownCleanly covers the server lifecycle end to end: it
// binds the port, serves, and returns without error once shutdown is
// requested — what a Spring Boot application gets from its embedded container.
// App.Run wires the same code to SIGINT and SIGTERM.
func TestRunServesAndShutsDownCleanly(t *testing.T) {
	app := NewApp("test", StackServlet)
	app.Log.SetOutput(io.Discard)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/ping",
		Handler: func(*Request) (ResponseEntity, error) { return OKText("pong"), nil }})
	app.Port = freePort(t)

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.RunContext(ctx) }()

	if body := getWithRetry(t, "http://127.0.0.1:"+strconv.Itoa(app.Port)+"/ping"); body != "pong" {
		t.Errorf("body = %q, want %q", body, "pong")
	}

	stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunContext returned %v, want a clean shutdown", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("RunContext did not return after the context was cancelled")
	}
}

// TestRunStopsOnSIGTERM covers App.Run itself, which is RunContext wired to
// SIGINT and SIGTERM. The signal is delivered to this process, so the test
// installs its own handler first to keep the default disposition from killing
// the test binary if Run's registration were ever dropped.
func TestRunStopsOnSIGTERM(t *testing.T) {
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGTERM)
	defer signal.Stop(guard)

	app := NewApp("test", StackServlet)
	app.Log.SetOutput(io.Discard)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/ping",
		Handler: func(*Request) (ResponseEntity, error) { return OKText("pong"), nil }})
	app.Port = freePort(t)

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	if body := getWithRetry(t, "http://127.0.0.1:"+strconv.Itoa(app.Port)+"/ping"); body != "pong" {
		t.Fatalf("body = %q, want %q", body, "pong")
	}
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("signalling: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v, want a clean shutdown", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Run did not return after SIGTERM")
	}
}

// TestRunReportsAnUnbindablePort covers the non-zero exit an application makes
// when it cannot bind its listener. The port is held on every interface,
// because RunContext binds ":port" and a listener on 127.0.0.1 alone would not
// conflict with it.
func TestRunReportsAnUnbindablePort(t *testing.T) {
	held, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("holding a port: %v", err)
	}
	defer func() { _ = held.Close() }()

	app := NewApp("test", StackServlet)
	app.Log.SetOutput(io.Discard)
	app.Port = held.Addr().(*net.TCPAddr).Port

	// A timeout, so a regression that lets the bind succeed fails the test
	// instead of hanging it.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = app.RunContext(ctx)
	if err == nil {
		t.Fatal("RunContext returned nil for a port that is already bound")
	}
	if ctx.Err() != nil {
		t.Fatal("RunContext bound a port that was already in use")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("releasing the reserved port: %v", err)
	}
	return port
}

func getWithRetry(t *testing.T, url string) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, err := http.Get(url)
		if err == nil {
			defer func() { _ = res.Body.Close() }()
			body, readErr := io.ReadAll(res.Body)
			if readErr != nil {
				t.Fatalf("reading the response: %v", readErr)
			}
			return strings.TrimSpace(string(body))
		}
		if time.Now().After(deadline) {
			t.Fatalf("the server never accepted a connection: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
