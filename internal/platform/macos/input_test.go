//go:build darwin && !ios

package macos

import (
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// TestMultilineEditorWrapsInItsViewport checks a multiline input's text
// view, its container and the text it lays out against the width the
// scroll view shows, as the input narrows and widens, with overlay and
// legacy scrollers (#42).
func TestMultilineEditorWrapsInItsViewport(t *testing.T) {
	onMain(func() { multilineEditorWraps(t) })
}

func multilineEditorWraps(t *testing.T) {
	loadAppKit()
	text := strings.Repeat("多行输入在可见区域内换行。", 20) + " https://example.com/" + strings.Repeat("a", 300)
	for _, style := range []int{1, 0} { // overlay, legacy
		window := objc.ID(apple.Class("NSWindow")).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(0, 0, 500, 200), uint(0), uint(2), false)
		window.Send(sel("setReleasedWhenClosed:"), false)
		scroll, tv := newMultilineEditor()
		window.Send(sel("contentView")).Send(selAddSubview, scroll)
		scroll.Send(sel("setScrollerStyle:"), style)
		tv.Send(sel("setString:"), apple.NSString(text))
		container := tv.Send(sel("textContainer"))
		layout := tv.Send(sel("layoutManager"))
		for _, width := range []float64{320, 185, 445, 60} {
			scroll.Send(sel("setFrame:"), apple.Rect(0, 0, width, 90))
			scroll.Send(sel("tile"))
			layout.Send(sel("ensureLayoutForTextContainer:"), container)
			visible := objc.Send[apple.CGSize](scroll, sel("contentSize")).Width
			editor := objc.Send[apple.CGRect](tv, sel("frame")).Size.Width
			wraps := objc.Send[apple.CGSize](container, sel("size")).Width
			used := objc.Send[apple.CGRect](layout, sel("usedRectForTextContainer:"), container)
			if editor != visible || wraps != visible || used.Origin.X+used.Size.Width > visible {
				t.Errorf("scroller style %d, %v pt wide: shows %v pt, editor %v pt, container %v pt, text to %v pt",
					style, width, visible, editor, wraps, used.Origin.X+used.Size.Width)
			}
		}
		scroll.Send(selRelease)
		window.Send(selRelease)
	}
}

// TestMultilineEditorLineHeightStylesTyping checks that an explicit line
// height reaches AppKit's typing attributes, which are what text inserted
// after the first layout inherits.
func TestMultilineEditorLineHeightStylesTyping(t *testing.T) {
	onMain(func() {
		loadAppKit()
		_, tv := newMultilineEditor()
		font := apple.Font(driver.Font{Size: 13})
		color := nsColor(driver.RGB(0x202020))
		setEditorParagraph(tv, objc.ID(font), color, 24, nsAlignCenter)
		attrs := tv.Send(sel("typingAttributes"))
		para := attrs.Send(sel("objectForKey:"), textAttrs().paragraph)
		if para == 0 {
			t.Fatal("typing attributes have no paragraph style")
		}
		if got := objc.Send[float64](para, sel("minimumLineHeight")); got != 24 {
			t.Errorf("minimum line height = %v, want 24", got)
		}
		if got := objc.Send[float64](para, sel("maximumLineHeight")); got != 24 {
			t.Errorf("maximum line height = %v, want 24", got)
		}
		if got := objc.Send[int](para, sel("alignment")); got != nsAlignCenter {
			t.Errorf("alignment = %d, want %d", got, nsAlignCenter)
		}
		apple.Release(font)
		tv.Send(selRelease)
	})
}
