package base_test

import (
	"strconv"
	"testing"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

// macDesktop gives the list tests the keyboard focus macOS gives pressable views.
var macDesktop = nsgotest.Environment(core.Environment{OS: core.OSMacOS, Idiom: core.IdiomDesktop})

// restyled is what a list's restyle must leave: the same scroll view at the same offset, the keyboard focus on the list and the keys moving its cursor.
type restyled struct {
	scroll   *nsgotest.View
	offset   core.Points
	tint     core.Color
	selector string
}

// check has the rows start at inset, the scroll view keep its ListStyle's tint, and everything else as it was.
func (r restyled) check(t *testing.T, app *nsgotest.App, inset core.Points, row string) {
	t.Helper()
	scroll := app.Find(r.selector).Children[0]
	if scroll.ID != r.scroll.ID {
		t.Fatalf("restyling remounted the rows: scroll view %d then %d", r.scroll.ID, scroll.ID)
	}
	if scroll.Style.Background != r.tint {
		t.Fatalf("the rows' background is %v, want ListStyle's %v under ListStyleFunc's", scroll.Style.Background, r.tint)
	}
	if got := leftIn(app.Find(row), scroll); got != inset {
		t.Fatalf("%s starts at %v, want ListStyleFunc's padding %v", row, got, inset)
	}
	if scroll.Offset.Y != r.offset {
		t.Fatalf("restyling moved the rows from %v to %v", r.offset, scroll.Offset.Y)
	}
	if !app.Find(r.selector).Focused {
		t.Fatal("restyling took the keyboard focus from the list")
	}
}

// leftIn is where v starts across scroll's content.
func leftIn(v, scroll *nsgotest.View) core.Points {
	var x core.Points
	for ; v != nil && v != scroll; v = v.Parent {
		x += v.Frame.Origin.X
	}
	return x
}

// TestListStyleFuncRestylesInPlace has ListStyleFunc merge over ListStyle as what it reads changes, keeping the list mounted: its scroll offset, the selection's cursor, the keyboard focus and the keys it holds.
func TestListStyleFuncRestylesInPlace(t *testing.T) {
	tint := core.RGB(0x123456)
	var inset *core.Signal[core.Points]
	var sel *base.Selection
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		inset = core.NewSignal(cx, core.Points(4))
		sel = base.NewSelection(cx, false)
		label := func(i int) string { return "row " + strconv.Itoa(i) }
		return base.List{TestID: "list", Label: "Rows", Count: func() int { return 40 }, Selection: sel, RowLabel: label,
			Row: func(_ *core.Scope, r core.ListRow) core.Node {
				return core.Text{ContentFunc: func() string { return label(r.Index()) }}
			},
			Style: core.Style{Height: core.Pt(120)}, ItemStyle: core.Style{Height: core.Pt(24)},
			ListStyle:     core.Style{Padding: core.All(2), Background: tint},
			ListStyleFunc: func() core.Style { return core.Style{Padding: core.All(inset.Get())} }}
	}, macDesktop)
	app.Press("label:row 1")
	app.Focus("id:list")
	if got := sel.Cursor(); got != 1 {
		t.Fatalf("the cursor is on %v, want 1", got)
	}
	app.Scroll("id:list", 200)
	r := restyled{scroll: app.Find("id:list").Children[0], tint: tint, selector: "id:list"}
	r.offset = r.scroll.Offset.Y
	r.check(t, app, 4, "label:row 10")

	inset.Set(16)
	app.Settle()

	r.check(t, app, 16, "label:row 10")
	if got := sel.Cursor(); got != 1 {
		t.Fatalf("restyling moved the cursor to %v", got)
	}
	app.Key("down")
	if got := sel.Cursor(); got != 2 {
		t.Fatalf("after restyling, Down moved the cursor to %v, want 2", got)
	}
}

// TestTreeListStyleFuncRestylesInPlace does for a Tree what TestListStyleFuncRestylesInPlace does for a List.
func TestTreeListStyleFuncRestylesInPlace(t *testing.T) {
	tint := core.RGB(0x123456)
	var inset *core.Signal[core.Points]
	items := make([]base.TreeItem, 40)
	for i := range items {
		items[i] = base.TreeItem{ID: strconv.Itoa(i), Label: "item " + strconv.Itoa(i)}
	}
	selected := ""
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		inset = core.NewSignal(cx, core.Points(4))
		return base.Tree{TestID: "tree", Label: "Items", Items: func() []base.TreeItem { return items },
			OnSelectedChange: func(id string) { selected = id },
			Style:            core.Style{Height: core.Pt(120)}, ItemStyle: core.Style{Height: core.Pt(24)},
			ListStyle:     core.Style{Padding: core.All(2), Background: tint},
			ListStyleFunc: func() core.Style { return core.Style{Padding: core.All(inset.Get())} }}
	}, macDesktop)
	app.Press("label:item 1")
	app.Focus("id:tree")
	if selected != "1" {
		t.Fatalf("item %q is selected, want 1", selected)
	}
	app.Scroll("id:tree", 200)
	r := restyled{scroll: app.Find("id:tree").Children[0], tint: tint, selector: "id:tree"}
	r.offset = r.scroll.Offset.Y
	r.check(t, app, 4, "label:item 10")

	inset.Set(16)
	app.Settle()

	r.check(t, app, 16, "label:item 10")
	app.Key("down")
	if selected != "2" {
		t.Fatalf("after restyling, Down selected %q, want 2", selected)
	}
}
