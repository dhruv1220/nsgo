package layout

import "testing"

// ahemSized measures text as ahemText does in a font k times its size: glyphs and lines 10k points.
func ahemSized(text string, k float32) *testNode {
	a := &ahemText{text: text}
	return &testNode{content: func(known Size[Opt], available Size[AvailableSpace]) Size[float32] {
		for _, o := range []*Opt{&known.Width, &known.Height} {
			o.Value /= k
		}
		for _, s := range []*AvailableSpace{&available.Width, &available.Height} {
			if s.Kind == SpaceDefinite {
				s.Value /= k
			}
		}
		m := a.measure(known, available)
		return Size[float32]{m.Width * k, m.Height * k}
	}}
}

// TestGridBaselineUsesFirstLineOfWrappedIntrinsicText is #280's history list: a title, a smaller status and a message per row. The leaves report no baseline, so each falls back to its height; measured at min-content width, the title and the message wrapped and took their wrapped heights as their baselines, pushing the row's other cells down. The message also wraps in its 1fr column, which must not move its row's first line either.
func TestGridBaselineUsesFirstLineOfWrappedIntrinsicText(t *testing.T) {
	type cell struct {
		text string
		k    float32
	}
	rows := [][3]cell{
		{{"Optimize\u200bapp\u200bdata", 1.5}, {"skipped", 1.2}, {"Safari\u200bis\u200brunning\u200bin\u200bthe\u200bbackground", 1.5}},
		{{"Flush\u200bDNS\u200bcache", 1.5}, {"done", 1.2}, {"Flushed", 1.5}},
		{{"Reset", 1.5}, {"ok", 1.2}, {"Reset", 1.5}},
	}
	var items []*testNode
	for _, row := range rows {
		for _, c := range row {
			items = append(items, ahemSized(c.text, c.k))
		}
	}
	root := newNode(Style{
		Display: DisplayGrid,
		Size:    Size[Length]{Width: Px(600), Height: Auto},
		GridTemplateColumns: []GridTemplateComponent{
			{Track: Track(MaxContent)},
			{Track: Track(MaxContent)},
			{Track: MinMax(Px(0), Fr(1))},
		},
		Gap:        Size[Length]{Width: Px(12), Height: Px(8)},
		AlignItems: alignOf(AlignBaseline),
	}, items...)

	computeLayout(root, definiteSpace(600))
	if got := root.children[2].layout().Size.Height; got != 30 {
		t.Fatalf("message height = %v, want 30: it should wrap to two lines in its column", got)
	}
	var prevBottom float32
	for r, row := range rows {
		top, bottom := inf32, float32(0)
		var firstLine [3]float32
		for c := range row {
			l := root.children[3*r+c].layout()
			firstLine[c] = l.Location.Y + 10*row[c].k
			top, bottom = fmin(top, l.Location.Y), fmax(bottom, l.Location.Y+l.Size.Height)
		}
		if firstLine[0] != firstLine[1] || firstLine[0] != firstLine[2] {
			t.Errorf("row %d: first lines end at %v, want one baseline", r, firstLine)
		}
		if r > 0 && top != prevBottom+8 {
			t.Errorf("row %d starts at %v, want %v: the row before ends at %v plus the gap", r, top, prevBottom+8, prevBottom)
		}
		prevBottom = bottom
	}
}
