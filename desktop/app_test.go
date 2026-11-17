package desktop_test

import (
	"testing"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestLaunchedAtLoginChoosesInitialView(t *testing.T) {
	for _, login := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "login"}[login], func(t *testing.T) {
			app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
				// Read once when the app first runs, as a tray app decides whether to open its welcome window.
				if desktop.LaunchedAtLogin(cx) {
					return core.Text{Content: "Background"}
				}
				return core.Text{Content: "Welcome"}
			}, nsgotest.Environment(core.Environment{LaunchedAtLogin: login}))
			if login {
				app.Find("text:Background")
			} else {
				app.Find("text:Welcome")
			}
		})
	}
}

// TestWindowWorkspacesFollowTheEnvironment checks that an app can tell whether SetWindowWorkspaces takes effect, as on Windows and Wayland, where it does not and the request still goes to the platform, and on macOS and X11, where it does.
func TestWindowWorkspacesFollowTheEnvironment(t *testing.T) {
	var cx *core.Scope
	app := nsgotest.Start(t, func(c *core.Scope) core.Node {
		cx = c
		text := func(s string) func(*core.Scope) core.Node {
			return func(*core.Scope) core.Node { return core.Text{Content: s} }
		}
		return core.Show{When: func() bool { return desktop.CanSetWindowWorkspaces(c) }, Then: text("Everywhere"), Else: text("Here")}
	}, nsgotest.Environment(core.Environment{OS: core.OSWindows, Idiom: core.IdiomDesktop}))
	app.Find("text:Here")
	cx.Post(func() { desktop.SetWindowWorkspaces(cx, desktop.WindowWorkspacesAll) })
	app.Settle()
	if n := len(app.Window); n == 0 || app.Window[n-1] != (core.WindowCommand{Action: core.WindowSetWorkspaces, Workspaces: desktop.WindowWorkspacesAll}) {
		t.Fatalf("window commands = %+v", app.Window)
	}
	app.SetEnvironment(core.Environment{OS: core.OSMacOS, Idiom: core.IdiomDesktop, CanSetWindowWorkspaces: true})
	app.Find("text:Everywhere")
}
