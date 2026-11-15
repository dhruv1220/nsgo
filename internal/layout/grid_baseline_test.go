package layout

import "testing"

func TestGridBaselineUsesFirstLineOfWrappedIntrinsicText(t *testing.T) {
	text := func(value string) *testNode {
		content := &ahemText{text: value}
		return &testNode{content: content.measure}
	}
	root := newNode(Style{
		Display: DisplayGrid,
		Size:    pxSize(600, 200),
		GridTemplateColumns: []GridTemplateComponent{
			{Track: Track(MaxContent)},
			{Track: Track(MaxContent)},
			{Track: MinMax(Px(0), Fr(1))},
		},
		Gap:        Size[Length]{Width: Px(12), Height: Px(8)},
		AlignItems: alignOf(AlignBaseline),
	},
		text("Optimize\u200bapp\u200bdata"), text("ok"), text("Safari\u200bis\u200brunning"),
		text("Flush\u200bDNS\u200bcache"), text("done"), text("Flushed"),
	)

	computeLayout(root, definiteSpace(600))
	a, b, c := root.children[0].layout(), root.children[1].layout(), root.children[2].layout()
	if a.Location.Y != b.Location.Y || a.Location.Y != c.Location.Y {
		t.Fatalf("first row baselines differ: a=%v b=%v c=%v", a, b, c)
	}
	d, e, f := root.children[3].layout(), root.children[4].layout(), root.children[5].layout()
	if d.Location.Y != e.Location.Y || d.Location.Y != f.Location.Y {
		t.Fatalf("second row baselines differ: d=%v e=%v f=%v", d, e, f)
	}
	if d.Location.Y < a.Location.Y+a.Size.Height+8 {
		t.Fatalf("rows overlap or lose their gap: first=%v second=%v", a, d)
	}
}
