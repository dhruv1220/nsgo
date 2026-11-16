package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// cdp is a connection to a browser over the Chrome DevTools protocol: one
// WebSocket, the sessions of the pages attached to it flattened into it.
type cdp struct {
	ws      *websocket.Conn
	mu      sync.Mutex
	next    int64
	pending map[int64]chan cdpMessage
	// on receives each event, on the reading goroutine.
	on func(cdpMessage)
	// browser is the browser's process.
	browser *exec.Cmd
	exited  <-chan struct{}
	profile string
}

type cdpMessage struct {
	ID        int64           `json:"id,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// browserPaths are where Chrome, Chromium and Edge install, by system.
var browserPaths = map[string][]string{
	"darwin": {
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	},
	"linux":   {"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"},
	"windows": {`C:\Program Files\Google\Chrome\Application\chrome.exe`, `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`},
}

// cdpOrigin is the origin the tool connects to the browser from.
const cdpOrigin = "http://127.0.0.1"

// findBrowser finds a Chromium browser: NSGO_BROWSER, else an installed
// Chrome, Chromium or Edge. NSGo never downloads one.
func findBrowser() (string, error) {
	if b := os.Getenv("NSGO_BROWSER"); b != "" {
		return b, nil
	}
	for _, p := range browserPaths[runtime.GOOS] {
		if path, err := exec.LookPath(p); err == nil {
			return path, nil
		}
	}
	return "", errors.New("the web target needs Chrome, Chromium or Edge: install one, or set NSGO_BROWSER to its executable")
}

// startBrowser starts a headless browser with a profile of its own and
// connects to it.
func startBrowser(ctx context.Context) (*cdp, error) {
	path, err := findBrowser()
	if err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", "nsgo-browser-")
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, "--headless=new", "--remote-debugging-port=0", "--user-data-dir="+profile,
		"--no-first-run", "--no-default-browser-check", "--disable-extensions", "--mute-audio",
		// The DevTools socket takes this tool, which names its origin.
		"--remote-allow-origins="+cdpOrigin,
		// A fine, hovering pointer, as Playwright gives it: headless finds
		// no input device and would measure as a touch screen.
		"--blink-settings=primaryHoverType=2,availableHoverTypes=2,primaryPointerType=4,availablePointerTypes=4",
		"about:blank")
	return startBrowserProcess(ctx, cmd, profile)
}

// startBrowserProcess owns the process and its profile, including a
// failed or cancelled launch. Wait runs once, from the start, so a dead
// browser is reported immediately and never left as an unreaped child.
func startBrowserProcess(ctx context.Context, cmd *exec.Cmd, profile string) (_ *cdp, err error) {
	var output browserOutput
	cmd.Stderr = &output
	// Browser children may inherit stderr; their pipe must not keep Wait
	// alive indefinitely after the browser itself has gone.
	cmd.WaitDelay = 3 * time.Second
	if err := cmd.Start(); err != nil {
		os.RemoveAll(profile)
		return nil, err
	}
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()
	defer func() {
		if err != nil {
			cmd.Process.Kill()
			<-exited
			os.RemoveAll(profile)
			if tail := strings.TrimSpace(string(output.data)); tail != "" {
				err = fmt.Errorf("%w\n%s", err, tail)
			}
		}
	}()
	startup, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	var url string
	for url == "" {
		data, err := os.ReadFile(filepath.Join(profile, "DevToolsActivePort"))
		if lines := strings.Split(strings.TrimSpace(string(data)), "\n"); err == nil && len(lines) == 2 {
			url = "ws://127.0.0.1:" + lines[0] + lines[1]
			break
		}
		select {
		case <-exited:
			if err := startup.Err(); err != nil {
				return nil, fmt.Errorf("waiting for the browser's DevTools port: %w", err)
			}
			return nil, fmt.Errorf("the browser exited before opening its DevTools port: %s", cmd.ProcessState)
		case <-startup.Done():
			return nil, fmt.Errorf("waiting for the browser's DevTools port: %w", startup.Err())
		case <-tick.C:
		}
	}
	config, err := websocket.NewConfig(url, cdpOrigin)
	if err != nil {
		return nil, err
	}
	ws, err := config.DialContext(startup)
	if err != nil {
		if startup.Err() != nil {
			return nil, fmt.Errorf("connecting to the browser's DevTools port: %w", startup.Err())
		}
		return nil, err
	}
	ws.MaxPayloadBytes = 256 << 20
	c := &cdp{ws: ws, pending: map[int64]chan cdpMessage{}, browser: cmd, exited: exited, profile: profile}
	go c.read()
	return c, nil
}

// browserOutput keeps only the last 64 KiB of stderr. exec.Cmd writes it
// on one goroutine; it is read only after Wait has joined that goroutine.
type browserOutput struct{ data []byte }

func (b *browserOutput) Write(p []byte) (int, error) {
	const limit = 64 << 10
	n := len(p)
	if len(p) >= limit {
		b.data = append(b.data[:0], p[len(p)-limit:]...)
	} else {
		if drop := len(b.data) + len(p) - limit; drop > 0 {
			b.data = append(b.data[:0], b.data[drop:]...)
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}

func (c *cdp) read() {
	for {
		var m cdpMessage
		if err := websocket.JSON.Receive(c.ws, &m); err != nil {
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}
		if m.ID != 0 {
			c.mu.Lock()
			ch := c.pending[m.ID]
			delete(c.pending, m.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- m
			}
			continue
		}
		c.mu.Lock()
		on := c.on
		c.mu.Unlock()
		if on != nil {
			on(m)
		}
	}
}

// call calls method in session ("" for the browser) and returns its
// result.
func (c *cdp) call(session, method string, params any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if params == nil {
		raw = []byte("{}")
	}
	c.mu.Lock()
	c.next++
	id := c.next
	ch := make(chan cdpMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()
	if err := websocket.JSON.Send(c.ws, cdpMessage{ID: id, SessionID: session, Method: method, Params: raw}); err != nil {
		return nil, err
	}
	select {
	case m, ok := <-ch:
		if !ok {
			return nil, errors.New("the browser closed")
		}
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("%s: no answer", method)
	}
}

func (c *cdp) close() {
	c.call("", "Browser.close", nil)
	c.ws.Close()
	select {
	case <-c.exited:
	case <-time.After(3 * time.Second):
		c.browser.Process.Kill()
		<-c.exited
	}
	os.RemoveAll(c.profile)
}
