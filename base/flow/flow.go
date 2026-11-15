// Package flow draws a directed diagram, such as a workflow, a dependency
// graph or a plan of tasks, and lays it out automatically: nodes fall into
// ranks along the edges' direction, ranks are ordered to avoid crossings
// and edges run between them as orthogonal lines. Applications build each
// node's content and own selection and what opening a node means; the view
// owns layout, routing and the camera.
package flow

import (
	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
)

// Direction is the way edges point.
type Direction uint8

const (
	// TopToBottom places sources at the top and edges pointing down.
	TopToBottom Direction = iota
	// LeftToRight places sources at the left and edges pointing right.
	LeftToRight
)

// Node is one card of the diagram. ID must be nonempty, unique and stable
// across changes; later duplicates are ignored. Label is its accessible
// name, the ID when empty.
//
// Content is the node's face, made when its ID first appears and kept
// while the ID stays, following the signals it reads. Positive Width and
// Height fix the node's size; otherwise the layout measures Content,
// laying the diagram out again when that size changes. Muted fades the
// node and its edges, as for skipped or inactive steps.
type Node struct {
	ID, Label     string
	Width, Height core.Points
	Muted         bool
	Content       core.Node
}

// Edge points from Source to Target. Edges to unknown nodes and self-loops
// are ignored; edges closing a cycle are drawn against the layout's
// direction. Label, when set, sits on the line. A zero Color takes the
// look's line and a zero Width 1.5 points.
type Edge struct {
	Source, Target string
	Label          string
	Dashed         bool
	Color          core.Color
	Width          core.Points
}

// Data is what the diagram shows; the view never modifies it.
type Data struct {
	Nodes []Node
	Edges []Edge
}

// Look colors what the view draws itself. Line is an edge's color and
// Emphasis that of the edges of the hovered or selected node; Label,
// LabelBackground and LabelBorder color an edge's label. Zero fields take
// the current text color, with lines, labels and the label's border faded,
// and a transparent label background. A styled form such as ui/flow supplies
// its own look.
type Look struct {
	Line, Emphasis                      core.Color
	Label, LabelBackground, LabelBorder core.Color
}

// View is the diagram, laid out at the size its layout takes, again when
// Data, read as a signal, changes; put it on a camera.View to pan and zoom
// it, as several diagrams can share one. Nodes take the keyboard focus in
// data order when the application can select or open them, and Camera,
// when set, follows the focused node.
//
// A press on a node selects it (OnSelect) and a double press opens it
// (OnOpen). Platforms hit-test views without a camera's transform, so on
// a camera the application hands the pointer to State's Press and Hover
// at points of the diagram, which find the node there themselves, and a
// press on empty space selects "".
type View struct {
	Data func() Data
	// Direction is the way edges point, read as a signal; nil is top to
	// bottom.
	Direction func() Direction
	// NodeGap separates nodes within a rank (40 when 0) and RankGap the
	// ranks (72 when 0).
	NodeGap, RankGap core.Points
	// Selected is the application's selected node, which the view rings
	// and whose edges it emphasizes.
	Selected func() string
	OnSelect func(id string)
	OnOpen   func(id string)
	// State holds measured sizes and the layout between mounts and takes
	// the pointer; nil keeps one of the view's own.
	State *State
	// Camera is the camera showing the diagram, which reveals the node the
	// keyboard focuses; the diagram's origin is the content's unless
	// Origin says where in it the diagram is.
	Camera *camera.State
	Origin func() core.Point
	// Empty replaces the diagram while it has no nodes, an em dash when
	// nil.
	Empty core.Node
	// Label describes the diagram for assistive technology, its count of
	// nodes and edges when empty.
	Label     string
	LabelFunc func() string
	LookFunc  func() Look
	// NodeClass, NodeStyle and NodeStyleFunc are each node's box, around
	// its Content, such as its focus ring; RingClass, RingStyle and
	// RingStyleFunc the ring around the selected node, 3 points outside
	// it, 2 points wide in the text's color unless they say otherwise.
	NodeClass     string
	NodeStyle     core.Style
	NodeStyleFunc func() core.Style
	RingClass     string
	RingStyle     core.Style
	RingStyleFunc func() core.Style
	Class         string
	Style         core.Style
	TestID        string
}

// State retains measured sizes and the layout between mounts, and takes
// the pointer for the view showing it. Its zero value is ready to use;
// keep one per diagram. Its methods run on the UI's goroutine and do
// nothing while no view shows it.
type State struct {
	v *view
	// What outlives a mount: measured sizes by ID.
	sizes map[string]core.Size
}

// Ready reports whether every node has been measured, laid out and shown.
func (s *State) Ready() bool { return s.v != nil && s.v.ready.GetUntracked() }

// Size is the size the diagram's layout takes, which a caller reading it
// follows.
func (s *State) Size() core.Size {
	if s.v == nil {
		return core.Size{}
	}
	s.v.laidOut.Get()
	return core.Size{Width: core.Points(s.v.result.width), Height: core.Points(s.v.result.height)}
}

// Position returns the top left corner of the node with id in diagram
// coordinates, and false if it is not laid out.
func (s *State) Position(id string) (x, y float32, ok bool) {
	b, ok := s.Bounds(id)
	return float32(b.Origin.X), float32(b.Origin.Y), ok
}

// Bounds is the frame of the node with id in diagram coordinates, false
// when it is not laid out.
func (s *State) Bounds(id string) (core.Bounds, bool) {
	if s.v == nil {
		return core.Bounds{}, false
	}
	i, ok := s.v.index[id]
	if !ok || i >= len(s.v.result.x) {
		return core.Bounds{}, false
	}
	w, h := s.v.nodes[i].size()
	return core.Bounds{Origin: core.Point{X: core.Points(s.v.result.x[i]), Y: core.Points(s.v.result.y[i])},
		Size: core.Size{Width: core.Points(w), Height: core.Points(h)}}, true
}

// Reveal has the view's Camera center the node with id, if it is shown.
func (s *State) Reveal(id string) {
	if s.v != nil {
		s.v.reveal(id, true)
	}
}

// NodeAt is the node at a point of the diagram, "" for none.
func (s *State) NodeAt(p core.Point) string {
	if s.v == nil {
		return ""
	}
	if i := s.v.hit(p); i >= 0 {
		return s.v.nodes[i].id
	}
	return ""
}

// Press presses the diagram at a point of it, as a camera hands it on:
// the node there is selected, and opened on a double press, or empty
// space selects "". It says a node took the press.
func (s *State) Press(p core.Point, e core.PressEvent) bool {
	if s.v == nil {
		return false
	}
	return s.v.press(p, e)
}

// Hover moves the pointer to a point of the diagram, emphasizing the
// edges of the node there, or off it when over is false.
func (s *State) Hover(p core.Point, over bool) {
	if s.v == nil {
		return
	}
	id := ""
	if over {
		id = s.NodeAt(p)
	}
	s.v.hovered.Set(id)
}

// Hovered is the node the pointer is over, "" for none, which a caller
// reading it follows, such as to show the arrow over nodes.
func (s *State) Hovered() string {
	if s.v == nil {
		return ""
	}
	return s.v.hovered.Get()
}
