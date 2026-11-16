//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// TestDiagonalCursors asks AppKit for the frame cursors with the values
// NSCursor.h gives them.
func TestDiagonalCursors(t *testing.T) {
	if _, err := purego.Dlopen(appKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		t.Fatal(err)
	}
	pointer := nsCursor(driver.CursorPointer)
	nwse, nesw := nsCursor(driver.CursorNwseResize), nsCursor(driver.CursorNeswResize)
	if nwse == 0 || nesw == 0 || nwse == nesw || nwse == pointer || nesw == pointer {
		t.Errorf("pointer %v, nwse %v, nesw %v", pointer, nwse, nesw)
	}
}

// TestSpanCursorGoesAheadOfItsText finds a span's cursor over its characters, and the text's own cursor over the others.
func TestSpanCursorGoesAheadOfItsText(t *testing.T) { onMain(func() { spanCursorGoesAheadOfItsText(t) }) }

func spanCursorGoesAheadOfItsText(t *testing.T) {
	loadAppKit()
	style := driver.TextStyle{Font: driver.Font{Size: 20}, Color: driver.RGB(0)}
	at := &driver.AttributedText{Text: "plain link", Style: style, Spans: []driver.TextSpan{
		{Start: 0, End: 6, Style: style},
		{Start: 6, End: 10, Style: style, Cursor: driver.CursorPointer},
	}}
	p := apple.TextEngine{}.Prepare(at)
	defer p.Release()
	size := p.Measure(driver.Auto)
	l := p.Layout(size).(*apple.Layout)
	defer l.Release()

	up := objc.ID(classView).Send(selNew)
	defer up.Send(selRelease)
	up.Send(selSetFrame, apple.Rect(0, 0, 400, 100))
	n := objc.ID(classView).Send(selNew)
	defer n.Send(selRelease)
	const left, top = 10, 20
	n.Send(selSetFrame, apple.Rect(left, top, float64(size.Width), float64(size.Height)))
	up.Send(selAddSubview, n)
	v := &view{kind: driver.KindText, native: n, content: n, style: driver.DefaultViewStyle, layout: l,
		box: driver.Bounds{Size: size}, spans: at.Spans}
	v.style.Cursor = driver.CursorHelp
	views[n] = v
	defer delete(views, n)

	// The window's point of (x, middle) in the text, NsgoView being flipped.
	cursorOver := func(x float64) (driver.Cursor, bool) {
		loc := objc.Send[apple.CGPoint](n, sel("convertPoint:toView:"), apple.CGPoint{X: x, Y: float64(size.Height) / 2}, objc.ID(0))
		return cursorAt(n, up, loc)
	}
	if c, own := cursorOver(2); c != driver.CursorHelp || !own {
		t.Errorf("over the plain span: %v, %v; want the text's help cursor", c, own)
	}
	if c, own := cursorOver(float64(size.Width) - 2); c != driver.CursorPointer || !own {
		t.Errorf("over the link: %v, %v; want the span's pointer", c, own)
	}
	v.style.Cursor = driver.CursorAuto
	if c, own := cursorOver(2); c != driver.CursorAuto || !own {
		t.Errorf("over the plain span of a text with no cursor: %v, %v; want auto", c, own)
	}
}
