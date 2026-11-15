// Package restart starts a desktop app again once it has exited, for desktop.Restart, and keeps an app that restarts as it launches from doing so forever.
package restart

import (
	"os"
	"sync/atomic"
	"time"
)

const guardEnv = "NSGO_RESTART_GUARD"

// LoopWindow is how soon after a restart another restart counts as a loop.
const LoopWindow = 10 * time.Second

// restartedAt is when markRestarted found this process started by Start, in Unix nanoseconds; 0 when it was not.
var restartedAt atomic.Int64

// markRestarted records that this process was started by Start, so a restart as it launches is refused in one log line instead of spawning forever. A restart the user asks for later, as after changing a setting, still runs.
func markRestarted() {
	if os.Getenv(guardEnv) != "" {
		restartedAt.Store(time.Now().UnixNano())
		os.Unsetenv(guardEnv)
	}
}

// Looping says whether this process, started by Start, is restarting again within LoopWindow of its start: an app restarting itself as it launches gets one visible refusal instead of chaining forever, while a restart the user asks for later still runs.
func Looping() bool {
	at := restartedAt.Load()
	return at != 0 && time.Since(time.Unix(0, at)) < LoopWindow
}
