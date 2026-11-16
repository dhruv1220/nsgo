//go:build ios && !nsgoembed && !nsgowatch

package backend

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/engine"
	"github.com/nsgo-dev/nsgo/internal/platform/ios"
)

func run(root engine.App, app func(*core.Scope), o Options) {
	updates := updateConfig()
	updates.Store = ios.InstalledBy()
	root, app = withUpdateChecks(root, app, updates, true)
	ios.Run(rootOf(root, app), ios.Options{ID: id, Components: o.Components, Fonts: o.Fonts, Appearance: o.Appearance, Perf: o.Monitor,
		Updates: updates})
}
