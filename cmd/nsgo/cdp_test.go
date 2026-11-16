package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// TestBrowserProcess is a child process controlled by the startup tests;
// it needs no installed browser and runs on every desktop system.
func TestBrowserProcess(t *testing.T) {
	mode := os.Getenv("NSGO_BROWSER_TEST_PROCESS")
	if mode == "" {
		return
	}
	profile := os.Getenv("NSGO_BROWSER_TEST_PROFILE")
	if mode == "fail" {
		fmt.Fprint(os.Stderr, strings.Repeat("x", 128<<10)+"\nsynthetic browser startup failure\n")
		os.Exit(7)
	}
	if err := os.WriteFile(filepath.Join(profile, "started"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if mode == "ready" {
		port := os.Getenv("NSGO_BROWSER_TEST_PORT")
		first, _, _ := strings.Cut(port, "\n")
		file := filepath.Join(profile, "DevToolsActivePort")
		if err := os.WriteFile(file, []byte(first+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		// A browser can create the port file before both lines arrive.
		time.Sleep(75 * time.Millisecond)
		if err := os.WriteFile(file, []byte(port+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		for {
			if _, err := os.Stat(filepath.Join(profile, "closed")); err == nil {
				os.Exit(0)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	for {
		time.Sleep(time.Hour)
	}
}

func browserTestProcess(t *testing.T, ctx context.Context, mode, profile string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestBrowserProcess$")
	cmd.Env = append(os.Environ(), "NSGO_BROWSER_TEST_PROCESS="+mode, "NSGO_BROWSER_TEST_PROFILE="+profile)
	return cmd
}

func browserReaped(t *testing.T, cmd *exec.Cmd, profile string) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Error("the browser was not waited for")
	}
	if _, err := os.Stat(profile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("browser profile remains: %v", err)
	}
}

func TestBrowserStartupReportsExit(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	profile := filepath.Join(t.TempDir(), "profile")
	if err := os.Mkdir(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := browserTestProcess(t, ctx, "fail", profile)
	_, err := startBrowserProcess(ctx, cmd, profile)
	if err == nil || !strings.Contains(err.Error(), "exit status 7") || !strings.Contains(err.Error(), "synthetic browser startup failure") {
		t.Fatalf("startup error = %v; want the exit status and stderr", err)
	}
	if len(err.Error()) > (64<<10)+256 {
		t.Errorf("startup error grew to %d bytes", len(err.Error()))
	}
	browserReaped(t, cmd, profile)
}

func TestBrowserStartupHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	profile := t.TempDir()
	cmd := browserTestProcess(t, ctx, "wait", profile)
	result := make(chan error, 1)
	go func() { _, err := startBrowserProcess(ctx, cmd, profile); result <- err }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(profile, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("startup = %v; want context.Canceled", err)
		}
		browserReaped(t, cmd, profile)
	case <-time.After(5 * time.Second):
		t.Fatal("startup kept waiting after cancellation")
	}
}

func TestBrowserStartupWaitsForTheCompletePortFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancel)
	profile := t.TempDir()
	srv := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var msg cdpMessage
			if err := websocket.JSON.Receive(ws, &msg); err != nil {
				return
			}
			if err := websocket.JSON.Send(ws, map[string]any{"id": msg.ID, "result": map[string]any{}}); err != nil {
				return
			}
			if msg.Method == "Browser.close" {
				os.WriteFile(filepath.Join(profile, "closed"), nil, 0o600)
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := browserTestProcess(t, ctx, "ready", profile)
	cmd.Env = append(cmd.Env, "NSGO_BROWSER_TEST_PORT="+port+"\n/devtools/browser/test")
	b, err := startBrowserProcess(ctx, cmd, profile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.close(); browserReaped(t, cmd, profile) })
	if _, err := b.call("", "Browser.getVersion", nil); err != nil {
		t.Fatal(err)
	}
}

func TestBrowserStartupCancelsTheHandshake(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	profile := t.TempDir()
	connected := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(connected)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := browserTestProcess(t, ctx, "ready", profile)
	cmd.Env = append(cmd.Env, "NSGO_BROWSER_TEST_PORT="+port+"\n/devtools/browser/test")
	result := make(chan error, 1)
	go func() { _, err := startBrowserProcess(ctx, cmd, profile); result <- err }()
	select {
	case <-connected:
	case <-time.After(10 * time.Second):
		t.Fatal("the browser handshake did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("startup = %v; want context.Canceled", err)
		}
		browserReaped(t, cmd, profile)
	case <-time.After(5 * time.Second):
		t.Fatal("the browser handshake kept waiting after cancellation")
	}
}
