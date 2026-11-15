//go:build !(darwin && !ios) && !(linux && !android) && !windows

package restart

import "errors"

// Start does nothing where the app does not start itself: phones and the web never return from nsgo.Run.
func Start([]string) error { return errors.New("restarting is for desktops") }

// Await does nothing: no process restarted the app.
func Await() {}
