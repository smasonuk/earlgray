// Package runtime implements the retained component tree and reconciliation.
package runtime

import (
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/layout"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/screen"
	"github.com/smason/earlgray/internal/style"
)

// Instance is a retained node in the component tree.
// It persists across renders to allow state to be preserved.
type Instance struct {
	// Descriptor of this node as of the last render.
	nd   *node.Node
	kind node.Kind // cached for quick comparison

	// Key for reconciliation (from nd.Key or position index).
	key string

	// Children instances, parallel to nd.Children after reconciliation.
	children []*Instance

	// Layout result from the most recent layout pass.
	layout layout.Result

	// Hook slots for UseState (index by hook call order).
	hookSlots []any
	hookIdx   int // reset to 0 at the start of each render

	// dirty marks this instance as needing re-render.
	dirty bool

	// compID is the identity of a component function (for ComponentKind nodes).
	compID uintptr
}

// Runtime manages the component tree lifecycle.
type Runtime struct {
	root  *Instance
	dirty bool
}

// New creates a new Runtime.
func New() *Runtime {
	rt := &Runtime{dirty: true}
	globalRuntime = rt
	return rt
}

// MarkDirty schedules a re-render.
func (r *Runtime) MarkDirty() {
	r.dirty = true
}

// IsDirty reports whether a re-render is needed.
func (r *Runtime) IsDirty() bool {
	return r.dirty
}

// Update reconciles the existing instance tree against a new node tree.
func (r *Runtime) Update(n *node.Node) {
	if r.root == nil {
		r.root = mount(n)
	} else {
		reconcile(r.root, n)
	}
	r.dirty = false
}

// RunLayout computes layout for the current instance tree.
func (r *Runtime) RunLayout(w, h int) {
	if r.root == nil {
		return
	}
	c := layout.Constraints{MinW: 0, MaxW: w, MinH: 0, MaxH: h}
	// Build a synthetic node tree from instances so that rendered component
	// output (which lives in the instance tree, not the raw node tree) is
	// visible to the layout engine.
	synth := buildSyntheticNode(r.root)
	t := layout.Layout(synth, c)
	applyLayout(r.root, t)
}

// buildSyntheticNode creates a node tree that mirrors the instance tree,
// substituting rendered component/keyed children so the layout engine can
// see the actual structure produced by component functions.
func buildSyntheticNode(inst *Instance) *node.Node {
	switch inst.nd.Kind {
	case node.ComponentKind:
		synth := &node.Node{Kind: node.ComponentKind, CompID: inst.nd.CompID}
		if len(inst.children) > 0 {
			synth.Children = []*node.Node{buildSyntheticNode(inst.children[0])}
		}
		return synth
	case node.KeyedKind:
		synth := &node.Node{Kind: node.KeyedKind, Key: inst.nd.Key}
		if len(inst.children) > 0 {
			synth.Children = []*node.Node{buildSyntheticNode(inst.children[0])}
		}
		return synth
	case node.ViewKind:
		synth := *inst.nd // copy value
		synth.Children = make([]*node.Node, len(inst.children))
		for i, child := range inst.children {
			synth.Children[i] = buildSyntheticNode(child)
		}
		return &synth
	default: // TextKind
		return inst.nd
	}
}

// Render paints the instance tree into buf.
func (r *Runtime) Render(buf *screen.Buffer) {
	if r.root == nil {
		return
	}
	renderInstance(r.root, buf)
}

// HandleEvent delivers a keyboard event to the runtime.
// Returns true if the event was consumed.
func (r *Runtime) HandleEvent(ev event.Event) bool {
	// Future: deliver to focused component.
	_ = ev
	return false
}

// mount creates a fresh Instance tree for the given node.
func mount(n *node.Node) *Instance {
	inst := &Instance{
		nd:   n,
		kind: n.Kind,
		key:  n.Key,
	}

	// For component nodes, render the component and mount its output.
	if n.Kind == node.ComponentKind {
		inst.compID = n.CompID
		child := renderComponent(inst, n)
		inst.children = []*Instance{mount(child)}
		return inst
	}

	// For keyed nodes, just wrap the child.
	if n.Kind == node.KeyedKind && len(n.Children) == 1 {
		inst.children = []*Instance{mount(n.Children[0])}
		return inst
	}

	// View or Text: mount children.
	inst.children = make([]*Instance, len(n.Children))
	for i, child := range n.Children {
		inst.children[i] = mount(child)
	}
	return inst
}

// reconcile updates an existing instance to match a new node descriptor.
func reconcile(inst *Instance, n *node.Node) {
	inst.nd = n

	// Component: re-render.
	if n.Kind == node.ComponentKind {
		if inst.compID != n.CompID {
			// Different component function — remount.
			*inst = *mount(n)
			return
		}
		child := renderComponent(inst, n)
		if len(inst.children) == 0 {
			inst.children = []*Instance{mount(child)}
		} else {
			reconcile(inst.children[0], child)
		}
		return
	}

	// Keyed node.
	if n.Kind == node.KeyedKind && len(n.Children) == 1 {
		if len(inst.children) == 0 {
			inst.children = []*Instance{mount(n.Children[0])}
		} else {
			reconcile(inst.children[0], n.Children[0])
		}
		return
	}

	// View/Text: reconcile children by key then position.
	reconcileChildren(inst, n.Children)
}

// reconcileChildren matches new children to existing instances by key+position.
func reconcileChildren(inst *Instance, newChildren []*node.Node) {
	old := inst.children

	// Build a map of keyed old instances.
	keyedOld := make(map[string]*Instance)
	for _, o := range old {
		if o.key != "" {
			keyedOld[o.key] = o
		}
	}

	next := make([]*Instance, len(newChildren))
	usedOld := make([]bool, len(old))

	for i, nc := range newChildren {
		var matched *Instance

		if nc.Key != "" {
			// Try keyed match.
			if o, ok := keyedOld[nc.Key]; ok {
				matched = o
			}
		} else {
			// Try positional match (same kind, no key).
			if i < len(old) && !usedOld[i] && old[i].key == "" && sameKind(old[i], nc) {
				matched = old[i]
				usedOld[i] = true
			}
		}

		if matched != nil {
			reconcile(matched, nc)
			next[i] = matched
		} else {
			next[i] = mount(nc)
		}
	}

	inst.children = next
}

// sameKind reports whether an instance and new node have the same identity.
func sameKind(inst *Instance, n *node.Node) bool {
	if inst.kind != n.Kind {
		return false
	}
	if n.Kind == node.ComponentKind {
		return inst.compID == n.CompID
	}
	return true
}

// renderComponent calls the component function with the instance as hook context.
func renderComponent(inst *Instance, n *node.Node) *node.Node {
	inst.hookIdx = 0
	prev := renderingInstance
	renderingInstance = inst
	defer func() { renderingInstance = prev }()
	return n.CompFn()
}

// applyLayout walks the layout tree and stores results in instances.
func applyLayout(inst *Instance, t *layout.Tree) {
	inst.layout = t.Result

	// For component/keyed nodes, the layout engine collapses them and returns
	// the child's layout tree directly. Apply t (not t.Children[0]) to the
	// rendered child instance.
	if inst.nd.Kind == node.ComponentKind || inst.nd.Kind == node.KeyedKind {
		if len(inst.children) > 0 {
			applyLayout(inst.children[0], t)
		}
		return
	}

	n := len(inst.children)
	if n > len(t.Children) {
		n = len(t.Children)
	}
	for i := 0; i < n; i++ {
		applyLayout(inst.children[i], t.Children[i])
	}
}

// renderInstance paints an instance into the buffer.
func renderInstance(inst *Instance, buf *screen.Buffer) {
	r := inst.layout.Rect
	content := inst.layout.Content

	if inst.nd.Kind == node.ViewKind {
		s := inst.nd.Style

		// Fill background.
		if s.Background.Kind != 0 || true { // always fill for clipping
			fillStyle := screen.CellStyle{
				Fg: s.Foreground,
				Bg: s.Background,
			}
			buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		}

		// Draw borders using box-drawing characters.
		borderStyle := screen.CellStyle{
			Fg:   s.Foreground,
			Bg:   s.Background,
			Bold: s.Bold,
		}
		drawBorders(buf, r, s.Border, borderStyle)

		// Recurse into children.
		for _, child := range inst.children {
			renderInstance(child, buf)
		}
		return
	}

	if inst.nd.Kind == node.TextKind {
		s := inst.nd.Style
		textStyle := screen.CellStyle{
			Fg:        s.Foreground,
			Bg:        s.Background,
			Bold:      s.Bold,
			Italic:    s.Italic,
			Underline: s.Underline,
		}
		buf.DrawTextClipped(
			content.X, content.Y,
			inst.nd.Text,
			textStyle,
			content.X, content.Y, content.W, content.H,
		)
		return
	}

	// Component or Keyed: render children.
	for _, child := range inst.children {
		renderInstance(child, buf)
	}
}

// drawBorders draws the border lines/corners for a rect.
func drawBorders(buf *screen.Buffer, r style.Rect, b style.Border, s screen.CellStyle) {
	if !b.Top && !b.Bottom && !b.Left && !b.Right {
		return
	}
	if r.W <= 0 || r.H <= 0 {
		return
	}

	const (
		horizontal = '─' // U+2500
		vertical   = '│' // U+2502
		tlCorner   = '┌' // U+250C
		trCorner   = '┐' // U+2510
		blCorner   = '└' // U+2514
		brCorner   = '┘' // U+2518
		tConnector = '┬' // T-junctions for partial borders
		bConnector = '┴'
		lConnector = '├'
		rConnector = '┤'
	)

	if b.Top {
		buf.DrawHLine(r.X, r.Y, r.W, horizontal, s)
	}
	if b.Bottom {
		buf.DrawHLine(r.X, r.Y+r.H-1, r.W, horizontal, s)
	}
	if b.Left {
		buf.DrawVLine(r.X, r.Y, r.H, vertical, s)
	}
	if b.Right {
		buf.DrawVLine(r.X+r.W-1, r.Y, r.H, vertical, s)
	}

	// Corners — only draw when two sides meet.
	if b.Top && b.Left {
		buf.SetCell(r.X, r.Y, tlCorner, s)
	}
	if b.Top && b.Right {
		buf.SetCell(r.X+r.W-1, r.Y, trCorner, s)
	}
	if b.Bottom && b.Left {
		buf.SetCell(r.X, r.Y+r.H-1, blCorner, s)
	}
	if b.Bottom && b.Right {
		buf.SetCell(r.X+r.W-1, r.Y+r.H-1, brCorner, s)
	}

	_ = tConnector
	_ = bConnector
	_ = lConnector
	_ = rConnector
}
