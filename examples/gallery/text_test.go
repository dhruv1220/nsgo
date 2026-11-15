package main

import (
	_ "embed"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

//go:embed resources/fonts/Pacifico-Regular.ttf
var pacifico []byte

// TestARegisteredFontMeasures registers a font as the app runs, under nsgo test -target, and lays out a text in it with its own glyphs, not the system font's that a family it cannot find falls back to (#313).
func TestARegisteredFontMeasures(t *testing.T) {
	if nsgotest.Target() == "" {
		t.Skip("a headless test's platform registers no fonts")
	}
	const words = "Registered fonts"
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		if err := cx.RegisterFont(pacifico); err != nil {
			t.Errorf("RegisterFont: %v", err)
		}
		return core.View{Style: core.Style{AlignItems: core.AlignStart}, Children: []core.Node{
			core.Text{TestID: "registered", Content: words, Style: core.Style{FontFamily: "Pacifico", FontSize: 20}},
			core.Text{TestID: "missing", Content: words, Style: core.Style{FontFamily: "NSGo Missing Family", FontSize: 20}},
		}}
	})
	size := func(id string) driver.Size { return app.Find("id:" + id).Frame.Size }
	// The web loads a face after RegisterFont returns.
	deadline := time.Now().Add(5 * time.Second)
	for app.Settle(); size("registered") == size("missing"); app.Settle() {
		if time.Now().After(deadline) {
			t.Fatalf("the registered font's text measures %v, as the fallback's does", size("registered"))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
