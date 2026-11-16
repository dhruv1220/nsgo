//go:build !darwin && !windows && !android && !linux && !(js && wasm)

package backend

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/engine"
)

func run(engine.App, func(*core.Scope), Options) {
	panic("nsgo: no backend for this platform yet")
}
