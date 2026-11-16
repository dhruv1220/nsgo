//go:build linux && !android

package linux

import (
	"fmt"
	"testing"
	"time"

	"github.com/ebitengine/purego"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/driver"
)

// widgetCursorName returns the name of the cursor set on a GTK widget, "" for none.
var widgetCursorName = func() func(uintptr) string {
	var get func(uintptr) uintptr
	var name func(uintptr) uintptr
	if openLibs() == nil {
		purego.RegisterLibFunc(&get, gtkLib, "gtk_widget_get_cursor")
		purego.RegisterLibFunc(&name, gtkLib, "gdk_cursor_get_name")
	}
	return func(widget uintptr) string {
		if c := get(widget); c != 0 {
			return goString(name(c))
		}
		return ""
	}
}()

// TestSpanCursorFollowsThePointer moves the pointer over a text whose second span names a cursor, changes that cursor under the still pointer and leaves, as GTK's motion controller reports it, and checks the cursor the text's widget shows.
func TestSpanCursorFollowsThePointer(t *testing.T) {
	if testing.Short() {
		t.Skip("opens a window")
	}
	title := "nsgo-native-" + t.Name()
	app := func(cx *core.Scope) core.Node {
		link := core.NewSignal(cx, driver.CursorPointer)
		cx.Effect(func() { cx.SetTitle(fmt.Sprintf("%s|%d", title, link.Get())) })
		return core.View{Style: core.Style{Width: driver.Pt(360), Height: driver.Pt(240), Padding: driver.All(10),
			Background: driver.RGB(0xffffff), FontSize: 20, Color: driver.RGB(0x000000)},
			Children: []core.Node{
				core.Text{Style: core.Style{Cursor: driver.CursorHelp}, Spans: []core.Span{
					{Content: "AAAAAAAA"},
					{Content: "BBBBBBBB", StyleFunc: func() core.Style { return core.Style{Cursor: link.Get()} }},
				}},
				core.View{TestID: "switch", OnPress: func(core.PressEvent) { link.Set(driver.CursorCrosshair) },
					Style: core.Style{Width: driver.Pt(40), Height: driver.Pt(40), Background: driver.RGB(0x0000ff)}},
			},
		}
	}
	w := startTestWindow(t, app, Options{Title: title, Width: 360, Height: 240})
	waitForTitle(t, w, fmt.Sprintf("%s|%d", title, driver.CursorPointer))

	var text *view
	var box, button, a, b driver.Bounds
	w.onLoop(t, func() {
		for _, v := range w.views {
			switch {
			case v == nil:
			case v.kind == driver.KindText:
				text = v
				sb, _ := w.surfaceBounds(v)
				box = driver.Bounds{Origin: driver.Point{X: sb.Origin.X + v.content.Origin.X, Y: sb.Origin.Y + v.content.Origin.Y}, Size: v.content.Size}
				if l, ok := v.text.(*textLayout); ok {
					a, _ = l.CaretRect(driver.Caret{Offset: 2})
					b, _ = l.CaretRect(driver.Caret{Offset: 12})
				}
			case v.props.TestID == "switch":
				button, _ = w.surfaceBounds(v)
			}
		}
	})
	if text == nil || b.Origin.X <= a.Origin.X || box.Size.Width < b.Origin.X+40 {
		t.Fatalf("text %v at %v, carets %v and %v: want a text wider than its glyphs", text, box, a, b)
	}
	y := float64(box.Origin.Y + box.Size.Height/2)
	plain, link := float64(box.Origin.X+a.Origin.X+2), float64(box.Origin.X+b.Origin.X+2)
	past := float64(box.Origin.X + box.Size.Width - 3) // right of the line's end
	shows := func(want string) {
		t.Helper()
		var got string
		deadline := time.Now().Add(5 * time.Second)
		for {
			w.onLoop(t, func() { got = widgetCursorName(text.widget) })
			if got == want || time.Now().After(deadline) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if got != want {
			t.Fatalf("the text's widget shows %q, want %q", got, want)
		}
	}
	move := func(x, y float64) { w.onLoop(t, func() { motionCallback(0, x, y, w.handle) }) }

	move(plain, y)
	shows("help")
	move(link, y)
	shows("pointer")
	move(past, y)
	shows("help")
	move(plain, y)
	shows("help")

	// The span's cursor changes under a still pointer.
	move(link, y)
	shows("pointer")
	w.onLoop(t, func() { w.press(driver.Point{X: button.Origin.X + 20, Y: button.Origin.Y + 20}) })
	waitForTitle(t, w, fmt.Sprintf("%s|%d", title, driver.CursorCrosshair))
	shows("crosshair")

	w.onLoop(t, func() { leaveCallback(0, w.handle) })
	shows("help")
}
