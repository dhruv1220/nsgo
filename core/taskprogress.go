package core

import (
	"math"
	"sync/atomic"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// TaskProgressState says what a task progress indicator shows.
type TaskProgressState uint8

const (
	// TaskProgressNone clears the indicator.
	TaskProgressNone TaskProgressState = iota
	// TaskProgressNormal shows Value as a completed fraction.
	TaskProgressNormal
	// TaskProgressIndeterminate shows that the task has no fraction yet.
	TaskProgressIndeterminate
	// TaskProgressPaused shows a paused task. Windows and macOS give it a paused color.
	TaskProgressPaused
	// TaskProgressError shows a task that failed. Windows and macOS give it an error color.
	TaskProgressError
)

// TaskProgress is how far a task has gone on the app's icon. Value is a
// fraction from 0 to 1 for Normal, Paused and Error; it is ignored for the
// other states.
type TaskProgress struct {
	State TaskProgressState
	Value float64
}

// taskProgressCalls counts the process's SetTaskProgress calls, for driver.TaskProgress.Call: the process's, not a root's, as the Dock tile and a launcher entry outlive an app that restarts in the same process.
var taskProgressCalls atomic.Uint64

// SetTaskProgress shows p on the taskbar button of the scope's window on Windows, or on the app's icon on macOS and Linux. On Windows, a scope whose surface has no button, such as the app's own scope where it opens its windows, a panel or a tray icon's view, sets the progress every window of the app shows while it sets none of its own. Android, iOS and the web ignore it. Where one indicator shows the progress of several windows, the latest call wins, whichever window's changes commit first. Repeating the value the same window sent last sends nothing, unless another window changed the progress since.
func (cx *Scope) SetTaskProgress(p TaskProgress) {
	t := cx.tree()
	if t == nil {
		return
	}
	state := p.State
	switch state {
	case TaskProgressNone, TaskProgressNormal, TaskProgressIndeterminate, TaskProgressPaused, TaskProgressError:
	default:
		state = TaskProgressNone
	}
	value := min(max(p.Value, 0), 1)
	if math.IsNaN(value) {
		value = 0
	}
	if state == TaskProgressNone || state == TaskProgressIndeterminate {
		value = 0
	}
	shown := driver.TaskProgress{State: driver.TaskProgressState(state), Value: value}
	rt := cx.rt
	if rt.taskProgressSent && rt.taskProgressFrom == t.id && rt.taskProgress == shown {
		return
	}
	rt.taskProgress, rt.taskProgressFrom, rt.taskProgressSent = shown, t.id, true
	next := shown
	next.Call = taskProgressCalls.Add(1)
	t.commands = append(t.commands, driver.MountItem{Kind: driver.ItemTaskProgress, Progress: &next})
	t.changed = true
}
