package main

import (
	"math"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestSwipeVelocityUsesTheAppClock(t *testing.T) {
	headless(t)
	var s swipe
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		s.r = newRing(cx)
		return core.View{}
	}, nsgotest.Environment(core.Environment{ReduceMotion: true}))
	app.Post(func() { s.wheel(core.WheelEvent{Touchpad: true, Phase: core.GestureBegan}) })
	app.Settle()
	app.Advance(10 * time.Millisecond)
	app.Post(func() {
		s.wheel(core.WheelEvent{Touchpad: true, Phase: core.GestureChanged, Delta: core.Point{X: -100}})
	})
	app.Settle()
	if math.Abs(s.speed+4000) > 0.001 {
		t.Fatalf("speed %g, want -4000 points/second after 10 ms", s.speed)
	}
	// A pause on the graph's clock expires the speed measured while dragging, however fast the test runs in real time.
	app.Advance(100 * time.Millisecond)
	app.Post(func() { s.wheel(core.WheelEvent{Touchpad: true, Phase: core.GestureEnded}) })
	app.Settle()
	if s.r.target != -step {
		t.Fatalf("stale velocity flung to %g, want %g", s.r.target, -step)
	}
}
