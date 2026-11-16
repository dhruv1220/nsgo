package driver

import "testing"

func TestSpanCursor(t *testing.T) {
	spans := []TextSpan{
		{Start: 0, End: 6},
		{Start: 6, End: 10, Cursor: CursorPointer},
		{Start: 10, End: 14, Cursor: CursorHelp, Hidden: true},
	}
	for _, tc := range []struct {
		i    int
		want Cursor
		ok   bool
	}{
		{-1, CursorAuto, false},
		{0, CursorAuto, false},
		{5, CursorAuto, false},
		{6, CursorPointer, true},
		{9, CursorPointer, true},
		{10, CursorAuto, false}, // hidden: its characters show nothing
		{14, CursorAuto, false},
	} {
		if c, ok := SpanCursor(spans, tc.i); c != tc.want || ok != tc.ok {
			t.Errorf("SpanCursor(%d) = %v, %v; want %v, %v", tc.i, c, ok, tc.want, tc.ok)
		}
	}
	if !HasSpanCursor(spans) || HasSpanCursor(spans[:1]) || HasSpanCursor(nil) {
		t.Error("HasSpanCursor does not tell the spans that name a cursor")
	}
}
