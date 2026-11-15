package driver

import "math"

// Points is a length in platform-independent points (iOS points, Android dp,
// CSS pixels at a zoom of 1).
type Points float32

// Auto marks a length the layout computes, such as a text engine's
// unbounded width. Styles use [Length] instead.
var Auto = Points(float32(math.NaN()))

// IsAuto reports whether p is Auto.
func (p Points) IsAuto() bool { return p != p }

// Point is a position in points.
type Point struct{ X, Y Points }

// Size is a width and a height in points.
type Size struct{ Width, Height Points }

// Bounds is a rectangle: its origin relative to the parent's and its size.
type Bounds struct {
	Origin Point
	Size   Size
}

// WindowPlacement is the side of an anchor rectangle where a panel goes.
// It is separate from core.Placement so the platform protocol does not
// depend on core.
type WindowPlacement uint8

const (
	WindowPlaceDefault WindowPlacement = iota
	WindowPlaceBottom
	WindowPlaceTop
	WindowPlaceLeft
	WindowPlaceRight
)

// PlaceWindow returns a panel's top-left corner and the side it used. It
// prefers side, flips to the opposite side when that one fits, and clamps
// the result to area when neither side fits.
func PlaceWindow(anchor Bounds, size Size, area Bounds, side WindowPlacement) (Point, WindowPlacement) {
	if side == WindowPlaceDefault {
		side = WindowPlaceBottom
	}
	opposite := func(p WindowPlacement) WindowPlacement {
		switch p {
		case WindowPlaceTop:
			return WindowPlaceBottom
		case WindowPlaceBottom:
			return WindowPlaceTop
		case WindowPlaceLeft:
			return WindowPlaceRight
		case WindowPlaceRight:
			return WindowPlaceLeft
		default:
			return WindowPlaceBottom
		}
	}
	point := func(p WindowPlacement) Point {
		const gap Points = 6
		switch p {
		case WindowPlaceTop:
			return Point{X: anchor.Origin.X + (anchor.Size.Width-size.Width)/2, Y: anchor.Origin.Y - size.Height - gap}
		case WindowPlaceLeft:
			return Point{X: anchor.Origin.X - size.Width - gap, Y: anchor.Origin.Y + (anchor.Size.Height-size.Height)/2}
		case WindowPlaceRight:
			return Point{X: anchor.Origin.X + anchor.Size.Width + gap, Y: anchor.Origin.Y + (anchor.Size.Height-size.Height)/2}
		default:
			return Point{X: anchor.Origin.X + (anchor.Size.Width-size.Width)/2, Y: anchor.Origin.Y + anchor.Size.Height + gap}
		}
	}
	fits := func(p Point) bool {
		return p.X >= area.Origin.X && p.Y >= area.Origin.Y &&
			p.X+size.Width <= area.Origin.X+area.Size.Width &&
			p.Y+size.Height <= area.Origin.Y+area.Size.Height
	}
	chosen := point(side)
	if !fits(chosen) {
		if flipped := point(opposite(side)); fits(flipped) {
			side, chosen = opposite(side), flipped
		}
	}
	maxX := area.Origin.X + max(area.Size.Width-size.Width, 0)
	maxY := area.Origin.Y + max(area.Size.Height-size.Height, 0)
	chosen.X = min(max(chosen.X, area.Origin.X), maxX)
	chosen.Y = min(max(chosen.Y, area.Origin.Y), maxY)
	return chosen, side
}

// DisplayInfo is one physical display in the desktop's screen coordinate space.
// Frame is its whole area; VisibleFrame is the area available to windows.
// Scale is device pixels per point, and Main marks the primary display.
type DisplayInfo struct {
	ID           string
	Frame        Bounds
	VisibleFrame Bounds
	Scale        float32
	Main         bool
}

// Contains reports whether p lies inside b.
func (b Bounds) Contains(p Point) bool {
	return p.X >= b.Origin.X && p.Y >= b.Origin.Y &&
		p.X < b.Origin.X+b.Size.Width && p.Y < b.Origin.Y+b.Size.Height
}

// Edges holds a length per side.
type Edges struct{ Top, Right, Bottom, Left Points }

// All returns edges of the same length on every side.
func All(p Points) Edges { return Edges{p, p, p, p} }

// Symmetric returns edges with vertical and horizontal lengths.
func Symmetric(vertical, horizontal Points) Edges {
	return Edges{vertical, horizontal, vertical, horizontal}
}

// Horizontal returns the left and right edges together.
func (e Edges) Horizontal() Points { return e.Left + e.Right }

// Vertical returns the top and bottom edges together.
func (e Edges) Vertical() Points { return e.Top + e.Bottom }

// Length is a style's length. Its zero value is auto: the layout decides.
type Length struct {
	value float32
	unit  unit
}

type unit uint8

const (
	unitAuto unit = iota
	unitPoints
	unitPercent
	unitMinContent
	unitMaxContent
	unitFitContent
	unitAutoMinimum
)

// The intrinsic sizes, for widths and heights: the narrowest a box gets
// without overflowing (its longest word), the widest its content asks for
// (no wrapping), and the widest it asks for up to the space it has.
var (
	MinContent = Length{unit: unitMinContent}
	MaxContent = Length{unit: unitMaxContent}
	FitContent = Length{unit: unitFitContent}
)

// AutoMinimum is CSS's automatic minimum, for a minimum size: a flex
// item keeps its content's size, where an unset minimum is 0, as in Yoga.
var AutoMinimum = Length{unit: unitAutoMinimum}

// Pt returns a length in points.
func Pt(p Points) Length { return Length{float32(p), unitPoints} }

// Pct returns a length in percent of the containing block's size.
func Pct(percent float32) Length { return Length{percent, unitPercent} }

// IsAuto reports whether l is auto.
func (l Length) IsAuto() bool { return l.unit == unitAuto }

// Points returns l's value in points, and whether l is a length in
// points.
func (l Length) Points() (Points, bool) {
	return Points(l.value), l.unit == unitPoints
}

// Percent returns l's value in percent, and whether l is a percentage.
func (l Length) Percent() (float32, bool) {
	return l.value, l.unit == unitPercent
}

// Resolve returns l in points against base, or Auto. A percentage of an
// unknown base is Auto.
func (l Length) Resolve(base Points) Points {
	switch l.unit {
	case unitPoints:
		return Points(l.value)
	case unitPercent:
		if base.IsAuto() {
			return Auto
		}
		return base * Points(l.value) / 100
	}
	return Auto
}

// Color is an sRGB color packed as 0xRRGGBBAA, the wire form of mount
// items. Its zero value is unset: no background, or text color inherited
// from the parent.
type Color uint32

// RGB returns an opaque color from 0xRRGGBB.
func RGB(hex uint32) Color { return Color(hex<<8 | 0xff) }

// RGBA returns a color from 0xRRGGBB and an alpha from 0 to 1.
func RGBA(hex uint32, alpha float32) Color {
	a := uint32(min(max(alpha, 0), 1)*255 + 0.5)
	return Color(hex<<8 | a)
}

// Faded returns c with its alpha multiplied by f, from 0 to 1.
func (c Color) Faded(f float32) Color {
	a := uint32(float32(c&0xff)*min(max(f, 0), 1) + 0.5)
	return c&^0xff | Color(a)
}

// Over returns c drawn over an opaque under.
func (c Color) Over(under Color) Color {
	a := uint32(c & 0xff)
	mix := func(shift uint) Color {
		top, bottom := uint32(c>>shift&0xff), uint32(under>>shift&0xff)
		return Color((top*a + bottom*(255-a) + 127) / 255 << shift)
	}
	return mix(24) | mix(16) | mix(8) | 0xff
}

// Components returns the red, green, blue and alpha channels from 0 to 1.
func (c Color) Components() (r, g, b, a float64) {
	return float64(c>>24&0xff) / 255, float64(c>>16&0xff) / 255,
		float64(c>>8&0xff) / 255, float64(c&0xff) / 255
}
