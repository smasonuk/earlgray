// Package runtime implements the retained component tree and reconciliation.
package runtime

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/layout"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/screen"
	"github.com/smason/earlgray/internal/style"
)

// Instance is a retained node in the component tree.
// It persists across renders to allow state to be preserved.
type Instance struct {
	id      uintptr   // stable identity; set on mount, preserved on reconcile
	parent  *Instance // parent in the instance tree; nil for root

	runtime *Runtime // owning runtime; used by UseState setters

	// Descriptor of this node as of the last render.
	nd   *node.Node
	kind node.Kind // cached for quick comparison

	// Key for reconciliation (from nd.Key or position index).
	key string

	// Children instances, parallel to nd.Children after reconciliation.
	children []*Instance

	// Layout result from the most recent layout pass.
	layout layout.Result

	// Hook slots for UseState (indexed by hook call order).
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

	nextID  uintptr   // incremented on each mount to assign stable IDs
	focused *Instance // currently focused instance (Focusable node), or nil
}

// New creates a new Runtime.
func New() *Runtime {
	return &Runtime{dirty: true}
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
		r.root = mount(r, nil, n)
	} else {
		r.root = reconcile(r, nil, r.root, n)
	}
	r.dirty = false
	r.ensureFocus()
}

// Focused returns the currently focused instance, or nil.
func (r *Runtime) Focused() *Instance {
	return r.focused
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
	renderInstance(r.root, buf, style.Style{})
}

// normalizeMod converts a tcell modifier mask to an input.Mod.
func normalizeMod(m tcell.ModMask) input.Mod {
	var out input.Mod
	if m&tcell.ModCtrl != 0 {
		out |= input.ModCtrl
	}
	if m&tcell.ModAlt != 0 {
		out |= input.ModAlt
	}
	if m&tcell.ModShift != 0 {
		out |= input.ModShift
	}
	return out
}

// HandleEvent delivers a keyboard event to the runtime.
// Returns true if the event was consumed.
func (r *Runtime) HandleEvent(ev event.Event) bool {
	if ev.Kind != event.KeyKind || r.root == nil {
		return false
	}

	// Tab moves focus forward through focusable nodes.
	if ev.Key.IsTab() {
		r.focusNext()
		r.MarkDirty()
		return true
	}

	// Deliver to focused node, then bubble up the parent chain.
	if r.focused != nil {
		press := input.KeyPress{
			Key:  event.NormalizeKey(ev.Key.Key, ev.Key.Rune),
			Rune: ev.Key.Rune,
			Mod:  normalizeMod(ev.Key.Mod),
		}
		for inst := r.focused; inst != nil; inst = inst.parent {
			if inst.nd != nil && inst.nd.OnKey != nil {
				if inst.nd.OnKey(press) {
					return true
				}
			}
		}
		return false
	}

	// No focused node: fall back to depth-first delivery.
	return deliverKey(r.root, ev.Key)
}

// deliverKey walks the instance tree depth-first (children before parent),
// calling the first OnKey handler that consumes the event.
// Used as fallback when nothing is focused.
func deliverKey(inst *Instance, key event.Key) bool {
	for _, child := range inst.children {
		if deliverKey(child, key) {
			return true
		}
	}
	if inst.nd.OnKey != nil {
		press := input.KeyPress{
			Key:  event.NormalizeKey(key.Key, key.Rune),
			Rune: key.Rune,
			Mod:  normalizeMod(key.Mod),
		}
		return inst.nd.OnKey(press)
	}
	return false
}

// collectFocusable returns all focusable instances in depth-first order.
func collectFocusable(root *Instance) []*Instance {
	var result []*Instance
	collectFocusableRec(root, &result)
	return result
}

func collectFocusableRec(inst *Instance, result *[]*Instance) {
	if inst.nd != nil && inst.nd.Focusable {
		*result = append(*result, inst)
	}
	for _, child := range inst.children {
		collectFocusableRec(child, result)
	}
}

// focusNext advances focus to the next focusable node, wrapping around.
func (r *Runtime) focusNext() {
	if r.root == nil {
		return
	}
	focusable := collectFocusable(r.root)
	if len(focusable) == 0 {
		r.focused = nil
		return
	}
	if r.focused == nil {
		r.focused = focusable[0]
		return
	}
	for i, inst := range focusable {
		if inst == r.focused {
			r.focused = focusable[(i+1)%len(focusable)]
			return
		}
	}
	// Focused instance no longer in tree; reset to first.
	r.focused = focusable[0]
}

// ensureFocus is called after each Update. If the focused instance was removed
// from the tree it moves focus to the first available focusable node. If
// nothing was focused and focusable nodes exist, it focuses the first one.
// Sets dirty if focus changed so the app can re-render to reflect focus state.
func (r *Runtime) ensureFocus() {
	if r.root == nil {
		r.focused = nil
		return
	}
	prev := r.focused
	focusable := collectFocusable(r.root)

	if r.focused != nil {
		for _, inst := range focusable {
			if inst == r.focused {
				return // still valid; no change
			}
		}
		// Previously focused instance is gone.
	}

	if len(focusable) > 0 {
		r.focused = focusable[0]
	} else {
		r.focused = nil
	}
	if r.focused != prev {
		r.dirty = true
	}
}

// mount creates a fresh Instance tree for the given node.
func mount(rt *Runtime, parent *Instance, n *node.Node) *Instance {
	rt.nextID++
	inst := &Instance{
		id:      rt.nextID,
		parent:  parent,
		runtime: rt,
		nd:      n,
		kind:    n.Kind,
		key:     n.Key,
	}

	// For component nodes, render the component and mount its output.
	if n.Kind == node.ComponentKind {
		inst.compID = n.CompID
		child := renderComponent(inst, n)
		inst.children = []*Instance{mount(rt, inst, child)}
		return inst
	}

	// For keyed nodes, just wrap the child.
	if n.Kind == node.KeyedKind && len(n.Children) == 1 {
		inst.children = []*Instance{mount(rt, inst, n.Children[0])}
		return inst
	}

	// View or Text: mount children.
	inst.children = make([]*Instance, len(n.Children))
	for i, child := range n.Children {
		inst.children[i] = mount(rt, inst, child)
	}
	return inst
}

// reconcile updates an existing instance to match a new node descriptor.
// Returns the instance to use — either inst (updated in place) or a fresh mount.
// parent is the parent of inst in the new tree (may be nil for root).
func reconcile(rt *Runtime, parent *Instance, inst *Instance, n *node.Node) *Instance {
	if !sameKind(inst, n) {
		return mount(rt, parent, n)
	}

	inst.nd = n
	inst.kind = n.Kind
	inst.key = n.Key

	switch n.Kind {
	case node.ComponentKind:
		child := renderComponent(inst, n)
		if len(inst.children) == 0 {
			inst.children = []*Instance{mount(rt, inst, child)}
		} else {
			inst.children[0] = reconcile(rt, inst, inst.children[0], child)
		}
	case node.KeyedKind:
		if len(n.Children) == 1 {
			if len(inst.children) == 0 {
				inst.children = []*Instance{mount(rt, inst, n.Children[0])}
			} else {
				inst.children[0] = reconcile(rt, inst, inst.children[0], n.Children[0])
			}
		}
	default:
		reconcileChildren(rt, inst, n.Children)
	}

	return inst
}

// reconcileChildren matches new children to existing instances by key+position.
func reconcileChildren(rt *Runtime, inst *Instance, newChildren []*node.Node) {
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
			next[i] = reconcile(rt, inst, matched, nc)
		} else {
			next[i] = mount(rt, inst, nc)
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

// renderInstance paints an instance into the buffer, inheriting color styles from parent.
func renderInstance(inst *Instance, buf *screen.Buffer, inherited style.Style) {
	r := inst.layout.Rect
	content := inst.layout.Content

	if inst.nd.Kind == node.ViewKind {
		s := style.Merge(inherited, inst.nd.Style)

		fillStyle := screen.CellStyle{
			Fg: s.Foreground,
			Bg: s.Background,
		}
		buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)

		borderStyle := screen.CellStyle{
			Fg:   s.Foreground,
			Bg:   s.Background,
			Bold: s.Bold,
		}
		drawBorders(buf, r, s.Border, borderStyle)

		for _, child := range inst.children {
			renderInstance(child, buf, s)
		}
		return
	}

	if inst.nd.Kind == node.TextKind {
		opts := inst.nd.TextOpts
		s := style.Merge(inherited, opts.Style)
		textStyle := screen.CellStyle{
			Fg:        s.Foreground,
			Bg:        s.Background,
			Bold:      s.Bold,
			Italic:    s.Italic,
			Underline: s.Underline,
		}
		drawMultilineText(buf, content, inst.nd.Text, textStyle, opts.Align)
		return
	}

	// Component or Keyed: render children.
	for _, child := range inst.children {
		renderInstance(child, buf, inherited)
	}
}

// alignedX computes the starting x for a single line of text given alignment.
func alignedX(rect style.Rect, line string, align node.TextAlign) int {
	width := runewidth.StringWidth(line)
	if width > rect.W {
		width = rect.W
	}
	switch align {
	case node.TextAlignCenter:
		return rect.X + (rect.W-width)/2
	case node.TextAlignRight:
		return rect.X + rect.W - width
	default:
		return rect.X
	}
}

// drawMultilineText renders text split by newlines with alignment support.
func drawMultilineText(buf *screen.Buffer, rect style.Rect, text string, st screen.CellStyle, align node.TextAlign) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i >= rect.H {
			break
		}
		x := alignedX(rect, line, align)
		y := rect.Y + i
		buf.DrawTextClipped(x, y, line, st, rect.X, rect.Y, rect.W, rect.H)
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
}
