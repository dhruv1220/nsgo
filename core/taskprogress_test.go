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
