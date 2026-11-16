//go:build ios && nsgowatch

package backend

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/engine"
	"github.com/nsgo-dev/nsgo/internal/platform/watchos"
)

func run(root engine.App, app func(*core.Scope), o Options) {
	watchos.Run(rootOf(root, app), watchos.Options{ID: id, Fonts: o.Fonts, Appearance: o.Appearance})
}
