//go:build linux && !android

package linux

import (
	"math"
	"os"
	"strings"
	"sync"

	"github.com/ebitengine/purego"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// GTK 4 deliberately leaves toplevel placement to a Wayland compositor.
// On X11 we can keep a non-activating panel at a global rectangle through
// GDK's X11 surface and Xlib, both loaded dynamically so Linux still builds
// and runs without X11 development files.
var (
	bindPanelX11 = sync.OnceFunc(func() {
		if os.Getenv("WAYLAND_DISPLAY") != "" || strings.Contains(strings.ToLower(os.Getenv("GDK_BACKEND")), "wayland") || os.Getenv("DISPLAY") == "" {
			return
		}
		for _, symbol := range []string{"gdk_x11_surface_get_xid"} {
			if _, err := purego.Dlsym(gtkLib, symbol); err != nil {
				return
			}
		}
		lib, err := purego.Dlopen("libX11.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			lib, err = purego.Dlopen("libX11.so", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		}
		if err != nil {
			return
		}
		for _, symbol := range []string{"XMoveWindow", "XMapWindow", "XRaiseWindow", "XFlush"} {
			if _, err := purego.Dlsym(lib, symbol); err != nil {
				return
			}
		}
		purego.RegisterLibFunc(&gdkX11SurfaceGetXID, gtkLib, "gdk_x11_surface_get_xid")
		purego.RegisterLibFunc(&x11MoveWindow, lib, "XMoveWindow")
		purego.RegisterLibFunc(&x11MapWindow, lib, "XMapWindow")
		purego.RegisterLibFunc(&x11RaiseWindow, lib, "XRaiseWindow")
		purego.RegisterLibFunc(&x11Flush, lib, "XFlush")
		x11PanelReady = true
	})
	gdkX11SurfaceGetXID func(uintptr) uintptr
	x11MoveWindow       func(display, window uintptr, x, y int32)
	x11MapWindow        func(display, window uintptr)
	x11RaiseWindow      func(display, window uintptr)
	x11Flush            func(display uintptr)
	x11PanelReady       bool
)

func panelX11Surface(w *window) (uintptr, uintptr, bool) {
	bindPanelX11()
	if !x11PanelReady || w.surface == 0 {
		return 0, 0, false
	}
	surface := gtkNativeGetSurface(gtkWidgetGetNative(w.surface))
	if surface == 0 {
		return 0, 0, false
	}
	xid := gdkX11SurfaceGetXID(surface)
	if xid == 0 {
		return 0, 0, false
	}
	return gdkDisplayGetDefault(), xid, true
}

func (w *window) placementFailed(message string) {
	if w.engine != nil {
		w.engine.Dispatch(driver.Event{Kind: driver.EventWindowPlacementFailed, Text: message})
	}
}

func (w *window) showPanel(anchorID uint32, anchor driver.Bounds, side driver.WindowPlacement, activate bool) {
	w.anchor, w.anchorBounds, w.windowPlacement = anchorID, anchor, side
	if !activate {
		// Map first so GDK creates the X11 surface and gives us its XID;
		// gtk_widget_set_visible does not request focus.
		gtkWidgetSetVisible(w.win, 1)
	}
	if anchor.Size.Width > 0 || anchor.Size.Height > 0 {
		if !w.placePanelAt(anchor, side) {
			w.visible = false
			gtkWidgetSetVisible(w.win, 0)
			w.placementFailed("Linux cannot place a panel at a screen rectangle on this display")
			return
		}
	}
	w.visible = true
	if activate {
		gtkWindowPresent(w.win)
		return
	}
	// Showing a GTK window as a widget avoids gtk_window_present's focus
	// request. X11 then maps and raises it explicitly without changing focus.
	gtkWidgetSetVisible(w.win, 1)
	if display, xid, ok := panelX11Surface(w); ok {
		x11MapWindow(display, xid)
		x11RaiseWindow(display, xid)
		x11Flush(display)
	}
}

func (w *window) placePanelAt(anchor driver.Bounds, side driver.WindowPlacement) bool {
	displays := displays()
	if len(displays) == 0 {
		return false
	}
	if _, _, ok := panelX11Surface(w); !ok {
		// A Wayland toplevel cannot honor a caret rectangle. Edge-docked
		// layer-shell support can be added without pretending this worked.
		return false
	}
	center := driver.Point{X: anchor.Origin.X + anchor.Size.Width/2, Y: anchor.Origin.Y + anchor.Size.Height/2}
	area := displays[0].VisibleFrame
	for _, d := range displays {
		if d.Frame.Contains(center) {
			area = d.VisibleFrame
			break
		}
	}
	var width, height int32
	gtkWindowGetDefaultSize(w.win, &width, &height)
	scale := gdkSurfaceGetScale(gtkNativeGetSurface(gtkWidgetGetNative(w.surface)))
	if scale <= 0 {
		scale = 1
	}
	size := driver.Size{Width: driver.Points(float64(width) / scale), Height: driver.Points(float64(height) / scale)}
	point, _ := driver.PlaceWindow(anchor, size, area, side)
	display, xid, ok := panelX11Surface(w)
	if !ok {
		return false
	}
	x11MoveWindow(display, xid, int32(math.Round(float64(point.X)*scale)), int32(math.Round(float64(point.Y)*scale)))
	x11Flush(display)
	return true
}
