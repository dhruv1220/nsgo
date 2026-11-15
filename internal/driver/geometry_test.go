package driver

import "testing"

func TestPlaceWindowFlipsAndClamps(t *testing.T) {
	area := Bounds{Origin: Point{X: 0, Y: 0}, Size: Size{Width: 400, Height: 300}}
	anchor := Bounds{Origin: Point{X: 100, Y: 260}, Size: Size{Width: 40, Height: 20}}
	point, side := PlaceWindow(anchor, Size{Width: 100, Height: 80}, area, WindowPlaceBottom)
	if side != WindowPlaceTop || point != (Point{X: 70, Y: 174}) {
		t.Fatalf("bottom placement = (%v, %v)", point, side)
	}
	anchor = Bounds{Origin: Point{X: -20, Y: 20}, Size: Size{Width: 10, Height: 10}}
	point, side = PlaceWindow(anchor, Size{Width: 100, Height: 80}, area, WindowPlaceLeft)
	if side != WindowPlaceLeft || point != (Point{X: 0, Y: 0}) {
		t.Fatalf("left placement = (%v, %v)", point, side)
	}
}
