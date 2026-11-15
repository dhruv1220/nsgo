package core

import (
	"slices"
	"testing"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

func TestKeyBindingsFireAsTheirOptionsSay(t *testing.T) {
	keymap := []driver.KeyBinding{
		{Key: "down", InText: !EditsText("down")},
		{Key: "mod+k", InText: !EditsText("mod+k")},
		{Key: "mod+z", InText: true, Repeat: true},
	}
	for _, c := range []struct {
		names           []string
		editing, repeat bool
		key             string
		consumed        bool
	}{
		{[]string{"down"}, false, false, "down", true},
		{[]string{"down"}, true, false, "", false}, // the field moves its caret
		{[]string{"mod+k", "meta+k"}, true, false, "mod+k", true},
		{[]string{"mod+k", "meta+k"}, false, true, "", true}, // held: consumed, fires once
		{[]string{"mod+z"}, true, true, "mod+z", true},
		{[]string{"x"}, false, false, "", false},
	} {
		key, consumed := driver.FindKey(keymap, c.names, c.editing, c.repeat)
		if key != c.key || consumed != c.consumed {
			t.Errorf("FindKey(%v, editing %v, repeat %v) = %q, %v; want %q, %v", c.names, c.editing, c.repeat,
				key, consumed, c.key, c.consumed)
		}
	}
}

func TestKeyLabelsFollowThePlatform(t *testing.T) {
	for _, c := range []struct {
		key   string
		os    OS
		apple bool
		want  string
	}{
		{"mod+shift+l", OSMacOS, true, "⇧⌘L"},
		{"mod+shift+l", OSWindows, false, "Ctrl+Shift+L"},
		{"ctrl+alt+escape", OSMacOS, true, "⌃⌥⎋"},
		{"meta+k", OSWindows, false, "Win+K"},
		{"meta+k", OSLinux, false, "Super+K"},
		{"f5", OSWeb, false, "F5"},
		{"mod+enter", OSWeb, true, "⌘↩"},
	} {
		if got := KeyLabel(c.key, c.os, c.apple); got != c.want {
			t.Errorf("KeyLabel(%q, %s) = %q, want %q", c.key, c.os, got, c.want)
		}
	}
}

func TestBackGoesToTheNewestHandlerThatTakesIt(t *testing.T) {
	var log []string
	var inner *Signal[bool]
	h := newHost(t, Size{Width: 100, Height: 100}, func(cx *Scope) Node {
		inner = NewSignal(cx, true)
		cx.OnBackPress(func() bool { log = append(log, "outer"); return true })
		return Show{When: inner.Get, Then: func(cx *Scope) Node {
			cx.OnBackPress(func() bool { log = append(log, "inner passes"); return false })
			return View{}
		}}
	})
	h.commit()
	h.tree.Dispatch(driver.Event{Kind: driver.EventBack})
	inner.Set(false)
	h.commit()
	h.tree.Dispatch(driver.Event{Kind: driver.EventBack})
	if want := []string{"inner passes", "outer", "outer"}; !slices.Equal(log, want) {
		t.Fatalf("log = %q, want %q", log, want)
	}
}

func TestReturnChoosesOnlyTheTopLayersDefault(t *testing.T) {
	for _, c := range []struct {
		name                string
		overlay, hasDefault bool
	}{{"modal", false, true}, {"modal-without-default", false, false}, {"overlay", true, true}, {"overlay-without-default", true, false}} {
		t.Run(c.name, func(t *testing.T) {
			var outer, inner *Signal[bool]
			var calls []string
			h := newHost(t, Size{Width: 300, Height: 400}, func(cx *Scope) Node {
				outer, inner = NewSignal(cx, true), NewSignal(cx, true)
				return Show{When: outer.Get, Then: func(*Scope) Node {
					return Modal{OnConfirm: func() { calls = append(calls, "outer") }, Children: []Node{
						Show{When: inner.Get, Then: func(*Scope) Node {
							var confirm func()
							if c.hasDefault {
								confirm = func() { calls = append(calls, "inner") }
							}
							if c.overlay {
								return Overlay{Role: RoleDialog, OnCancel: func() {}, OnConfirm: confirm}
							}
							return Modal{OnConfirm: confirm}
						}},
					}}
				}}
			})
			checkKey := func(want bool) {
				t.Helper()
				key, _ := driver.FindKey(h.tree.keymap(), []string{"enter"}, false, false)
				if (key == "enter") != want {
					t.Fatalf("Return binding %q, want active %v", key, want)
				}
			}
			h.commit()
			checkKey(c.hasDefault)
			h.tree.Dispatch(driver.Event{Kind: driver.EventKey, Text: "enter"})
			if c.hasDefault && !slices.Equal(calls, []string{"inner"}) || !c.hasDefault && len(calls) != 0 {
				t.Fatalf("top layer confirmation called %v", calls)
			}
			calls = nil
			inner.Set(false)
			b := h.commit()
			if kinds(b)[driver.ItemKeymap] != 1 {
				t.Fatal("removing top layer did not refresh keyboard bindings")
			}
			checkKey(true)
			h.tree.Dispatch(driver.Event{Kind: driver.EventKey, Text: "enter"})
			if !slices.Equal(calls, []string{"outer"}) {
				t.Fatalf("outer confirmation called %v", calls)
			}
			outer.Set(false)
			h.commit()
			checkKey(false)
		})
	}
}
