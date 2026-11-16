package driver

import "testing"

func TestPlaceWindow(t *testing.T) {
	area := Bounds{Origin: Point{X: 0, Y: 25}, Size: Size{Width: 400, Height: 300}}
	panel := Size{Width: 100, Height: 80}
	for _, c := range []struct {
		name   string
		anchor Bounds
		side   WindowPlacement
		want   Point
		took   WindowPlacement
	}{
		{"below by default", Bounds{Origin: Point{X: 100, Y: 60}, Size: Size{Width: 40, Height: 20}}, WindowPlaceDefault, Point{X: 70, Y: 86}, WindowPlaceBottom},
		{"above", Bounds{Origin: Point{X: 100, Y: 200}, Size: Size{Width: 40, Height: 20}}, WindowPlaceTop, Point{X: 70, Y: 114}, WindowPlaceTop},
		{"left", Bounds{Origin: Point{X: 200, Y: 100}, Size: Size{Width: 2, Height: 20}}, WindowPlaceLeft, Point{X: 94, Y: 70}, WindowPlaceLeft},
		{"right", Bounds{Origin: Point{X: 100, Y: 100}, Size: Size{Width: 2, Height: 20}}, WindowPlaceRight, Point{X: 108, Y: 70}, WindowPlaceRight},
		{"a point", Bounds{Origin: Point{X: 200, Y: 100}}, WindowPlaceBottom, Point{X: 150, Y: 106}, WindowPlaceBottom},
		{"flipped above at the bottom", Bounds{Origin: Point{X: 100, Y: 285}, Size: Size{Width: 40, Height: 20}}, WindowPlaceBottom, Point{X: 70, Y: 199}, WindowPlaceTop},
		{"flipped below at the top", Bounds{Origin: Point{X: 100, Y: 40}, Size: Size{Width: 40, Height: 20}}, WindowPlaceTop, Point{X: 70, Y: 66}, WindowPlaceBottom},
		{"flipped right at the left edge", Bounds{Origin: Point{X: 20, Y: 100}, Size: Size{Width: 2, Height: 20}}, WindowPlaceLeft, Point{X: 28, Y: 70}, WindowPlaceRight},
		// In a corner the side flips across and is clamped along: the panel must not cover the caret.
		{"flipped left in the top-right corner", Bounds{Origin: Point{X: 360, Y: 30}, Size: Size{Width: 2, Height: 20}}, WindowPlaceRight, Point{X: 254, Y: 25}, WindowPlaceLeft},
		{"clamped along the side", Bounds{Origin: Point{X: 380, Y: 100}, Size: Size{Width: 10, Height: 10}}, WindowPlaceBottom, Point{X: 300, Y: 116}, WindowPlaceBottom},
		{"clamped when neither side fits", Bounds{Origin: Point{X: -20, Y: 20}, Size: Size{Width: 10, Height: 10}}, WindowPlaceLeft, Point{X: 0, Y: 25}, WindowPlaceLeft},
		{"an unknown side is below", Bounds{Origin: Point{X: 100, Y: 60}, Size: Size{Width: 40, Height: 20}}, WindowPlacement(9), Point{X: 70, Y: 86}, WindowPlaceBottom},
	} {
		if got, took := PlaceWindow(c.anchor, panel, area, c.side); got != c.want || took != c.took {
			t.Errorf("%s: placed at %v on side %d, want %v on side %d", c.name, got, took, c.want, c.took)
		}
	}
	// A panel larger than the area keeps its top-left corner in it.
	if got, _ := PlaceWindow(Bounds{Origin: Point{X: 100, Y: 100}}, Size{Width: 500, Height: 400}, area, WindowPlaceBottom); got != area.Origin {
		t.Errorf("a panel larger than the area is at %v, want %v", got, area.Origin)
	}
}
