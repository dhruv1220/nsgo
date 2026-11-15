package core

import (
	"testing"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// TestTaskProgressRepeatsWhatAnotherTreeChanged checks that the repeat filter is the app's, not a surface's: the Dock tile and a Linux launcher entry show one progress for the whole app, so a value a tree sent before is sent again once another tree changed it.
func TestTaskProgressRepeatsWhatAnotherTreeChanged(t *testing.T) {
	var app, inWindow *Scope
	var win *Window
	w := newWindows(t, func(cx *Scope) {
		app = cx
		cx.SetShutdownMode(ShutdownOnExplicit)
		win = cx.OpenWindow(WindowOptions{Title: "Downloads"}, func(cx *Scope) Node {
			inWindow = cx
			return View{}
		})
	})
	id := win.t.ID()
	if id == 0 || inWindow == nil || inWindow.tree() != win.t || app.tree() != w.app {
		t.Fatalf("window tree %d, scopes %v %v", id, inWindow, app)
	}
	w.items = map[uint32][]driver.MountItem{}
	sent := func(tree uint32) []driver.TaskProgress {
		var out []driver.TaskProgress
		for _, it := range w.take(tree, driver.ItemTaskProgress) {
			out = append(out, *it.Progress)
		}
		return out
	}
	busy := TaskProgress{State: TaskProgressIndeterminate}
	inWindow.SetTaskProgress(busy)
	inWindow.SetTaskProgress(busy)
	w.commit()
	if got := sent(id); len(got) != 1 || got[0].State != driver.TaskProgressIndeterminate {
		t.Fatalf("window sent %+v, want one indeterminate", got)
	}
	app.SetTaskProgress(TaskProgress{})
	w.commit()
	if got := sent(0); len(got) != 1 || got[0].State != driver.TaskProgressNone {
		t.Fatalf("app sent %+v, want one clear", got)
	}
	inWindow.SetTaskProgress(busy)
	w.commit()
	if got := sent(id); len(got) != 1 || got[0].State != driver.TaskProgressIndeterminate {
		t.Errorf("after the app cleared it the window sent %+v, want indeterminate again", got)
	}
}

// TestTaskProgressNumbersCallsInTheirOrder checks that a call's number follows the calls, not the commits: the app's tree commits before its windows', so a window's progress set before the app's clear in the same pass reaches the platform after it, which keeps the clear as the later call (driver.TaskProgress.Call).
func TestTaskProgressNumbersCallsInTheirOrder(t *testing.T) {
	var app, inWindow *Scope
	var win *Window
	w := newWindows(t, func(cx *Scope) {
		app = cx
		cx.SetShutdownMode(ShutdownOnExplicit)
		win = cx.OpenWindow(WindowOptions{Title: "Downloads"}, func(cx *Scope) Node {
			inWindow = cx
			return View{}
		})
	})
	id := win.t.ID()
	w.items = map[uint32][]driver.MountItem{}
	inWindow.SetTaskProgress(TaskProgress{State: TaskProgressNormal, Value: 1})
	app.SetTaskProgress(TaskProgress{})
	w.commit()
	fromApp, fromWindow := w.take(0, driver.ItemTaskProgress), w.take(id, driver.ItemTaskProgress)
	if len(fromApp) != 1 || len(fromWindow) != 1 {
		t.Fatalf("the app sent %d items and the window %d, want one each", len(fromApp), len(fromWindow))
	}
	if cleared, full := fromApp[0].Progress, fromWindow[0].Progress; cleared.State != driver.TaskProgressNone || full.State != driver.TaskProgressNormal || full.Call == 0 || cleared.Call <= full.Call {
		t.Fatalf("the app sent %+v and the window %+v, want the app's clear the later call", *cleared, *full)
	}
	// The app's clear is the progress shown, so clearing it again sends nothing, and the window's value is sent again.
	app.SetTaskProgress(TaskProgress{})
	inWindow.SetTaskProgress(TaskProgress{State: TaskProgressNormal, Value: 1})
	w.commit()
	if got := w.take(0, driver.ItemTaskProgress); len(got) != 0 {
		t.Errorf("the app sent %d items for a repeated clear, want none", len(got))
	}
	if got := w.take(id, driver.ItemTaskProgress); len(got) != 1 || got[0].Progress.State != driver.TaskProgressNormal {
		t.Errorf("the window sent %d items after the app cleared its progress, want one", len(got))
	}
}
