//go:build windows

package windows

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/nsgo-dev/nsgo/core"
)

// TestInputSources selects another enabled keyboard layout and restores
// the one it replaced; an unknown or disabled one changes nothing.
func TestInputSources(t *testing.T) {
	scopes := make(chan *core.Scope, 1)
	hwnd := startWith(t, func(c *core.Scope) core.Node {
		scopes <- c
		return core.View{}
	}, Options{Title: "nsgo-input-sources|", Width: 320, Height: 200})
	cx := <-scopes
	var start string
	var others []string
	onMain(t, hwnd, func(w *window) {
		start = currentInputSource()
		n, _, _ := getKeyboardLayoutList.Call(0, 0)
		hkls := make([]uintptr, n)
		if n > 0 {
			getKeyboardLayoutList.Call(n, uintptr(unsafe.Pointer(&hkls[0])))
		}
		// Each enabled layout's ID, as it reads once active.
		for _, hkl := range hkls {
			if was, _, _ := activateKeyboardLayout.Call(hkl, 0); was != 0 {
				if id := currentInputSource(); id != start {
					others = append(others, id)
				}
				activateKeyboardLayout.Call(was, 0)
			}
		}
	})
	current := func() string {
		var id string
		onMain(t, hwnd, func(*window) { id = currentInputSource() })
		return id
	}
	waitFor := func(want string) {
		t.Helper()
		for deadline := time.Now().Add(3 * time.Second); current() != want; {
			if time.Now().After(deadline) {
				t.Fatalf("layout %q, want %q", current(), want)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	cx.Post(func() { cx.SelectInputSource("0000ffff") })
	time.Sleep(200 * time.Millisecond)
	waitFor(start)
	cx.Post(cx.RestoreInputSource)
	if len(others) == 0 {
		t.Logf("only %s is enabled: nothing to switch to", start)
		return
	}
	cx.Post(func() { cx.SelectInputSource(others[0]) })
	waitFor(others[0])
	cx.Post(cx.RestoreInputSource)
	waitFor(start)
}

// TestLifecycle reports the app active or inactive while its window shows, and in the background and minimized while it is minimized.
func TestLifecycle(t *testing.T) {
	scopes := make(chan *core.Scope, 1)
	hwnd := startWith(t, func(c *core.Scope) core.Node {
		scopes <- c
		return core.View{}
	}, Options{Title: "nsgo-lifecycle|", Width: 320, Height: 200})
	cx := <-scopes
	state := func() core.Lifecycle {
		got := make(chan core.Lifecycle, 1)
		cx.Post(func() { got <- core.Untracked(cx, cx.Env().Lifecycle) })
		return <-got
	}
	waitFor := func(ok func(core.Lifecycle) bool, what string) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); !ok(state()); {
			if time.Now().After(deadline) {
				t.Fatalf("lifecycle %v, want %s", state(), what)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	minimized := func() bool {
		got := make(chan bool, 1)
		cx.Post(func() { got <- core.Untracked(cx, cx.Env().Minimized) })
		return <-got
	}
	waitFor(func(l core.Lifecycle) bool { return l != core.LifecycleBackground }, "shown")
	if minimized() {
		t.Fatal("a shown window reads minimized")
	}
	onMain(t, hwnd, func(w *window) { showWindow.Call(w.hwnd, swMinimize) })
	waitFor(func(l core.Lifecycle) bool { return l == core.LifecycleBackground }, "background")
	if !minimized() {
		t.Fatal("a minimized window does not read minimized")
	}
	onMain(t, hwnd, func(w *window) { showWindow.Call(w.hwnd, swRestore) })
	waitFor(func(l core.Lifecycle) bool { return l != core.LifecycleBackground }, "shown again")
	if minimized() {
		t.Fatal("a restored window still reads minimized")
	}
}

// TestAppLifecycleFollowsItsWindows runs an app the way nsgo.Run runs an app function, on a hidden root window, with two windows: minimizing one turns only that window's content background and minimized, and the app's own scope turns background only once neither shows.
func TestAppLifecycleFollowsItsWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("opens windows")
	}
	const title = "nsgo-app-lifecycle"
	appScope := make(chan *core.Scope, 1)
	first, second := make(chan *core.Scope, 1), make(chan *core.Scope, 1)
	content := func(scopes chan *core.Scope) func(*core.Scope) core.Node {
		return func(cx *core.Scope) core.Node {
			select {
			case scopes <- cx:
			default:
			}
			return core.View{}
		}
	}
	app := core.AppRoot(func(cx *core.Scope) {
		appScope <- cx
		cx.OpenWindow(core.WindowOptions{Title: title + "-first", Width: 240, Height: 160}, content(first))
		cx.OpenWindow(core.WindowOptions{Title: title + "-second", Width: 240, Height: 160}, content(second))
	})
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer close(done)
		Run(app, Options{Title: title, Width: 240, Height: 160, Hidden: true})
	}()
	root := waitForWindow(t, title, done)
	t.Cleanup(func() {
		postMessageW.Call(root, wmClose, 0, 0)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("the app did not close")
		}
	})
	firstWindow, secondWindow := waitForWindow(t, title+"-first", done), waitForWindow(t, title+"-second", done)
	cxApp, cxFirst, cxSecond := <-appScope, <-first, <-second
	type state struct {
		lifecycle core.Lifecycle
		minimized bool
	}
	read := func(cx *core.Scope) state {
		got := make(chan state, 1)
		cx.Post(func() {
			got <- core.Untracked(cx, func() state { return state{cx.Env().Lifecycle(), cx.Env().Minimized()} })
		})
		return <-got
	}
	shown := func(s state) bool { return s.lifecycle != core.LifecycleBackground && !s.minimized }
	hidden := func(s state) bool { return s.lifecycle == core.LifecycleBackground && s.minimized }
	waitFor := func(what string, ok func() bool) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); !ok(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("%s: app %+v, first %+v, second %+v", what, read(cxApp), read(cxFirst), read(cxSecond))
			}
		}
	}
	waitFor("the windows did not show", func() bool {
		return read(cxApp).lifecycle != core.LifecycleBackground && shown(read(cxFirst)) && shown(read(cxSecond))
	})
	showWindow.Call(firstWindow, swMinimize)
	waitFor("the first window did not minimize alone", func() bool {
		return hidden(read(cxFirst)) && shown(read(cxSecond)) && read(cxApp).lifecycle != core.LifecycleBackground
	})
	showWindow.Call(secondWindow, swMinimize)
	waitFor("the app stayed in use with both windows minimized", func() bool {
		return hidden(read(cxSecond)) && read(cxApp).lifecycle == core.LifecycleBackground
	})
	showWindow.Call(firstWindow, swRestore)
	waitFor("the app did not come back with its first window", func() bool {
		return shown(read(cxFirst)) && hidden(read(cxSecond)) && read(cxApp).lifecycle != core.LifecycleBackground
	})
	if read(cxApp).minimized {
		t.Error("the app's own scope, in no window, reads minimized")
	}
}
