//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

var (
	taskProgressClassOnce sync.Once
	classTaskProgressBar  objc.Class
	dockProgressContainer objc.ID
	dockProgressBar       objc.ID
	dockProgressState     driver.TaskProgress
)

func registerTaskProgressClass() {
	taskProgressClassOnce.Do(func() {
		classTaskProgressBar = register("NsgoTaskProgressBar", apple.Class("NSView"), nil, []objc.MethodDef{
			{Cmd: sel("isFlipped"), Fn: func(objc.ID, objc.SEL) bool { return true }},
			{Cmd: sel("drawRect:"), Fn: drawTaskProgressBar},
		})
	})
}

// setDockProgress keeps the app icon visible and draws a small progress bar
// across its lower edge. The Dock tile is a snapshot, so indeterminate is a
// static half-filled bar; callers get a live indicator once a fraction is
// available.
func setDockProgress(progress driver.TaskProgress) {
	app := objc.ID(apple.Class("NSApplication")).Send(sel("sharedApplication"))
	tile := app.Send(sel("dockTile"))
	if progress.State == driver.TaskProgressNone {
		if dockProgressContainer != 0 {
			tile.Send(sel("setContentView:"), objc.ID(0))
			dockProgressContainer.Send(selRelease)
			dockProgressContainer, dockProgressBar = 0, 0
			tile.Send(sel("display"))
		}
		return
	}
	if dockProgressBar != 0 {
		dockProgressState = progress
		dockProgressBar.Send(selSetNeedsDisplay, true)
		tile.Send(sel("display"))
		return
	}
	registerTaskProgressClass()
	size := objc.Send[apple.CGSize](tile, sel("size"))
	frame := apple.Rect(0, 0, size.Width, size.Height)
	container := objc.ID(apple.Class("NSView")).Send(selAlloc).Send(sel("initWithFrame:"), frame)
	icon := objc.ID(apple.Class("NSImageView")).Send(selAlloc).Send(sel("initWithFrame:"), frame)
	icon.Send(sel("setImage:"), app.Send(sel("applicationIconImage")))
	icon.Send(sel("setImageScaling:"), 3) // NSImageScaleProportionallyUpOrDown.
	container.Send(selAddSubview, icon)
	icon.Send(selRelease)
	bar := objc.ID(classTaskProgressBar).Send(selAlloc).Send(sel("initWithFrame:"), apple.Rect(8, 4, size.Width-16, 8))
	dockProgressState = progress
	dockProgressContainer, dockProgressBar = container, bar
	container.Send(selAddSubview, bar)
	bar.Send(selRelease)
	tile.Send(sel("setContentView:"), container)
	tile.Send(sel("display"))
}

func drawTaskProgressBar(self objc.ID, _ objc.SEL, _ apple.CGRect) {
	context := objc.ID(apple.Class("NSGraphicsContext")).Send(selCurrentContext)
	if context == 0 {
		return
	}
	ctx := objc.Send[uintptr](context, selCGContext)
	bounds := objc.Send[apple.CGRect](self, selBounds)
	track := apple.CGColor(driver.RGBA(0x777777, .55))
	apple.FillRects(ctx, []apple.CGRect{bounds}, track)
	apple.Release(track)
	value := dockProgressState.Value
	if dockProgressState.State == driver.TaskProgressIndeterminate {
		value = .5
	}
	if value <= 0 {
		return
	}
	color := driver.RGB(0x30d158)
	switch dockProgressState.State {
	case driver.TaskProgressPaused:
		color = driver.RGB(0xff9f0a)
	case driver.TaskProgressError:
		color = driver.RGB(0xff453a)
	}
	fill := apple.CGColor(color)
	apple.FillRects(ctx, []apple.CGRect{apple.Rect(0, 0, bounds.Size.Width*value, bounds.Size.Height)}, fill)
	apple.Release(fill)
}
