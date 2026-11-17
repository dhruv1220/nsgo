package desktop_test

import (
	"fmt"
	"slices"
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

func TestFrontmostAppFollowsTheAppInFront(t *testing.T) {
	editor := desktop.FrontmostAppInfo{ID: "com.example.editor", Name: "Editor", PID: 41}
	player := desktop.FrontmostAppInfo{ID: "com.example.player", Name: "Player", PID: 42, FullScreen: true}
	var seen []desktop.FrontmostAppInfo
	var can []bool
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		cx.Effect(func() { seen = append(seen, desktop.FrontmostApp(cx)) })
		cx.Effect(func() { can = append(can, desktop.CanReadFrontmostApp(cx)) })
		return core.View{}
	}, nsgotest.Environment(core.Environment{FrontmostApp: editor, CanReadFrontmostApp: true}))
	app.SetEnvironment(core.Environment{FrontmostApp: player, CanReadFrontmostApp: true})
	// Another change to the environment leaves the app in front alone.
	app.SetEnvironment(core.Environment{FrontmostApp: player, CanReadFrontmostApp: true, Dark: true})
	// As on Wayland: a platform that does not follow the app in front reports none, whatever it holds.
	app.SetEnvironment(core.Environment{FrontmostApp: player})
	if want := []desktop.FrontmostAppInfo{editor, player, {}}; !slices.Equal(seen, want) {
		t.Fatalf("frontmost apps %v, want %v", seen, want)
	}
	if want := []bool{true, false}; !slices.Equal(can, want) {
		t.Fatalf("CanReadFrontmostApp %v, want %v", can, want)
	}
}

// refuseAll is the Access of code that may reach nothing.
type refuseAll struct{}

func (refuseAll) Allows(core.AccessRequest) bool { return false }
func (refuseAll) WebHosts() []string             { return nil }

// onlyFrontmost allows the frontmost app alone.
type onlyFrontmost struct{}

func (onlyFrontmost) Allows(req core.AccessRequest) bool {
	return req.Permission == core.AccessFrontmostApp
}
func (onlyFrontmost) WebHosts() []string { return nil }

func TestFrontmostAppIsHeldToAccess(t *testing.T) {
	for _, tc := range []struct {
		name   string
		access core.Access
		want   string
	}{
		{"refused", refuseAll{}, "text:none false"},
		{"granted", onlyFrontmost{}, "text:Editor true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
				cx.ProvideAccess(tc.access)
				name := desktop.FrontmostApp(cx).Name
				if name == "" {
					name = "none"
				}
				return core.Text{Content: fmt.Sprintf("%s %v", name, desktop.CanReadFrontmostApp(cx))}
			}, nsgotest.Environment(core.Environment{FrontmostApp: desktop.FrontmostAppInfo{ID: "com.example.editor", Name: "Editor"}, CanReadFrontmostApp: true}))
			app.Find(tc.want)
		})
	}
}
