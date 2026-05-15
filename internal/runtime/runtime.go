// Package runtime implements the retained component tree and reconciliation.
package runtime

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/smasonuk/earlgray/internal/event"
	"github.com/smasonuk/earlgray/internal/input"
	"github.com/smasonuk/earlgray/internal/layout"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/screen"
	"github.com/smasonuk/earlgray/internal/style"
)

// Instance is a retained node in the component tree.
// It persists across renders to allow state to be preserved.
type Instance struct {
	id     uintptr   // stable identity; set on mount, preserved on reconcile
	parent *Instance // parent in the instance tree; nil for root

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

	// Hook slots for component hooks (indexed by hook call order).
	hookSlots []hookSlot
	hookIdx   int // reset to 0 at the start of each render

	// dirty marks this instance as needing re-render.
	dirty bool

	// compID is the identity of a component function (for ComponentKind nodes).
	compID uintptr

	// Scroll state for scrollable nodes such as TextPanelKind.
	// scrollY is measured in visual lines after wrapping.
	// scrollX is measured in terminal cells and is only used when word wrap is disabled.
	scrollX            int
	scrollY            int
	textPanelResetKey  string
	textPanelScrollSet bool

	// Cursor state for editable text areas.
	textAreaCursor int

	// textAreaSelectionAnchor is the fixed end of the current text selection.
	// -1 means no active selection. When >= 0, the selected range is
	// [min(anchor, cursor), max(anchor, cursor)).
	textAreaSelectionAnchor int

	// textAreaDragging is true while the user is drag-selecting with the mouse.
	textAreaDragging bool

	// Whether the next textarea render should force the cursor into view.
	// Mouse-wheel scrolling intentionally clears this, so users can scroll
	// away from the cursor without the view snapping back.
	textAreaEnsureCursorVisible bool
}

// cursorState holds the cursor position requested by the most recently rendered
// node with CursorVisible == true.
type cursorState struct {
	visible bool
	x, y    int
}

// focusScopeFrame tracks one level of focus scope nesting.
type focusScopeFrame struct {
	scope     *Instance // the scope instance
	restoreTo *Instance // the focused instance before this scope was entered
}

type hookKind int

const (
	hookState hookKind = iota
	hookEffect
	hookRef
)

type hookSlot struct {
	kind   hookKind
	state  any
	effect effectSlot
}

type effectSlot struct {
	deps    []any
	cleanup func()
}

type pendingEffect struct {
	inst   *Instance
	idx    int
	effect func() func()
}

// Runtime manages the component tree lifecycle.
type Runtime struct {
	root  *Instance
	dirty bool

	nextID  uintptr     // incremented on each mount to assign stable IDs
	focused *Instance   // currently focused instance (Focusable node), or nil
	cursor  cursorState // cursor position requested during the last Render

	// scopeStack tracks nested focus scopes. Each frame records the scope instance
	// and what was focused before entering that scope.
	scopeStack []focusScopeFrame

	pendingEffects []pendingEffect
	appCtx         AppContext

	lastMouseButtons input.MouseButton
}

// AppContext provides app-level actions to components.
type AppContext struct {
	Post  func(func())
	Quit  func()
	Every func(time.Duration, func()) func()
}

// New creates a new Runtime.
func New() *Runtime {
	return &Runtime{dirty: true}
}

// MarkDirty schedules a re-render.
func (r *Runtime) MarkDirty() {
	r.dirty = true
}

// SetAppContext configures the app context for UseApp.
func (r *Runtime) SetAppContext(ctx AppContext) {
	r.appCtx = ctx
}

// GetAppContext returns the current app context.
func (r *Runtime) GetAppContext() AppContext {
	return r.appCtx
}

// SetPost configures an optional callback used to marshal work onto the app loop.
// Deprecated: use SetAppContext.
func (r *Runtime) SetPost(post func(func())) {
	r.appCtx.Post = post
}

// IsDirty reports whether a re-render is needed.
func (r *Runtime) IsDirty() bool {
	return r.dirty
}

// Update reconciles the existing instance tree against a new node tree.
func (r *Runtime) Update(n *node.Node) {
	r.pendingEffects = nil
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

// Cursor returns the cursor position requested during the last Render call.
// visible is false if no node requested a cursor.
func (r *Runtime) Cursor() (x, y int, visible bool) {
	return r.cursor.x, r.cursor.y, r.cursor.visible
}

func (r *Runtime) enqueue(fn func()) {
	if fn == nil {
		return
	}
	if r.appCtx.Post != nil {
		r.appCtx.Post(fn)
		return
	}
	fn()
}

func (r *Runtime) enqueueEffect(inst *Instance, idx int, effect func() func()) {
	r.pendingEffects = append(r.pendingEffects, pendingEffect{
		inst:   inst,
		idx:    idx,
		effect: effect,
	})
}

// RunEffects executes effects queued during the most recent committed render.
func (r *Runtime) RunEffects() {
	pending := r.pendingEffects
	r.pendingEffects = nil

	for _, p := range pending {
		if !isInstanceMounted(r.root, p.inst) {
			continue
		}
		if p.idx < 0 || p.idx >= len(p.inst.hookSlots) {
			continue
		}

		slot := &p.inst.hookSlots[p.idx]
		if slot.kind != hookEffect {
			continue
		}

		if slot.effect.cleanup != nil {
			slot.effect.cleanup()
			slot.effect.cleanup = nil
		}

		cleanup := p.effect()
		if cleanup != nil && isInstanceMounted(r.root, p.inst) {
			slot.effect.cleanup = cleanup
		}
	}
}

// Dispose unmounts the current instance tree and runs any outstanding cleanups.
func (r *Runtime) Dispose() {
	if r.root != nil {
		unmount(r.root)
	}
	r.root = nil
	r.focused = nil
	r.scopeStack = nil
	r.pendingEffects = nil
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
	case node.ViewKind, node.OverlayKind:
		synth := *inst.nd // copy value
		synth.Children = make([]*node.Node, len(inst.children))
		for i, child := range inst.children {
			synth.Children[i] = buildSyntheticNode(child)
		}
		return &synth
	default: // TextKind, RichTextKind, TextPanelKind, TextAreaKind, ScrollableListKind
		return inst.nd
	}
}

// Render paints the instance tree into buf.
func (r *Runtime) Render(buf *screen.Buffer) {
	r.cursor = cursorState{}
	if r.root == nil {
		return
	}
	renderInstance(r.root, buf, style.Style{}, &r.cursor)
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

// HandleEvent delivers an event to the runtime.
// Returns true if the event was consumed.
func (r *Runtime) HandleEvent(ev event.Event) bool {
	if r.root == nil {
		return false
	}

	if ev.Kind == event.PasteKind {
		if r.handlePaste(ev.Paste.Text) {
			r.MarkDirty()
			return true
		}
		return false
	}

	if ev.Kind == event.MouseKind {
		return r.handleMouse(ev.Mouse)
	}

	if ev.Kind != event.KeyKind {
		return false
	}

	press := input.KeyPress{
		Key:  event.NormalizeKey(ev.Key.Key, ev.Key.Rune, ev.Key.Mod),
		Rune: ev.Key.Rune,
		Mod:  normalizeMod(ev.Key.Mod),
	}
	root := activeFocusRoot(r.root)

	if r.focused != nil {
		for _, inst := range focusedPath(root, r.focused) {
			if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnKeyCapture != nil {
				if inst.nd.OnKeyCapture(press) {
					return true
				}
			}
		}
	} else if deliverKeyCapture(root, ev.Key) {
		return true
	}

	// Shift+Tab moves focus backward; plain Tab moves forward.
	if ev.Key.IsShiftTab() {
		r.focusPrev()
		r.MarkDirty()
		return true
	}
	if ev.Key.IsTab() {
		r.focusNext()
		r.MarkDirty()
		return true
	}

	// Deliver to focused node, then bubble up the parent chain.
	if r.focused != nil {
		if r.focused.nd != nil &&
			r.focused.nd.Kind == node.TextPanelKind &&
			!r.focused.nd.Disabled &&
			handleTextPanelKey(r.focused, press) {
			r.MarkDirty()
			return true
		}

		if r.focused.nd != nil &&
			r.focused.nd.Kind == node.TextAreaKind &&
			!r.focused.nd.Disabled &&
			handleTextAreaKey(r.focused, press) {
			r.MarkDirty()
			return true
		}

		if r.focused.nd != nil &&
			r.focused.nd.Kind == node.ScrollableListKind &&
			!r.focused.nd.Disabled &&
			handleScrollableListKey(r.focused, press) {
			r.MarkDirty()
			return true
		}

		for inst := r.focused; inst != nil; inst = inst.parent {
			if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnKey != nil {
				if inst.nd.OnKey(press) {
					return true
				}
			}
			if inst == root {
				break
			}
		}
		return false
	}

	// No focused node: fall back to depth-first delivery within the active focus scope.
	return deliverKey(root, ev.Key)
}

func (r *Runtime) handlePaste(text string) bool {
	if text == "" {
		return false
	}
	if r.focused == nil {
		return false
	}
	if r.focused.nd != nil && r.focused.nd.Kind == node.TextAreaKind && !r.focused.nd.Disabled {
		return handleTextAreaPaste(r.focused, text)
	}
	root := activeFocusRoot(r.root)
	for inst := r.focused; inst != nil; inst = inst.parent {
		if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnPaste != nil {
			if inst.nd.OnPaste(text) {
				return true
			}
		}
		if inst == root {
			break
		}
	}
	return false
}

// handleMouse dispatches a mouse event: hit-tests, focuses clicked nodes,
// scrolls TextPanels on wheel events, and bubbles OnMouse handlers.
func (r *Runtime) handleMouse(m event.Mouse) bool {
	action := input.ActionMotion
	if m.Button != r.lastMouseButtons {
		if (m.Button & ^r.lastMouseButtons) != 0 {
			action = input.ActionPress
		} else {
			action = input.ActionRelease
		}
		r.lastMouseButtons = m.Button
	}

	focusRoot := activeFocusRoot(r.root)
	hit := hitTest(focusRoot, m.X, m.Y)
	if hit == nil {
		return false
	}

	consumed := false

	// Left click: focus the nearest focusable ancestor of the hit node.
	// hitTest returns the deepest node (often a Text leaf); walking up finds the widget.
	if action == input.ActionPress && m.Button&input.MouseLeft != 0 {
		if target := nearestFocusableAncestor(hit, focusRoot); target != nil {
			if r.focused != target {
				r.focused = target
				r.dirty = true
				consumed = true
			}
		}

		for ta := hit; ta != nil; ta = ta.parent {
			if ta.nd != nil && ta.nd.Kind == node.TextAreaKind && !ta.nd.Disabled {
				if handleTextAreaClick(ta, m.X-ta.layout.Content.X, m.Y-ta.layout.Content.Y) {
					r.MarkDirty()
					consumed = true
				}
				break
			}
			if ta.nd != nil && ta.nd.Kind == node.ScrollableListKind && !ta.nd.Disabled {
				if handleScrollableListClick(ta, m.Y-ta.layout.Content.Y) {
					r.MarkDirty()
					consumed = true
				}
				break
			}
			if ta == focusRoot {
				break
			}
		}
	}

	// Left motion: extend textarea drag selection.
	if action == input.ActionMotion && m.Button&input.MouseLeft != 0 {
		if r.focused != nil && r.focused.nd != nil &&
			r.focused.nd.Kind == node.TextAreaKind && !r.focused.nd.Disabled &&
			r.focused.textAreaDragging {
			localX := m.X - r.focused.layout.Content.X
			localY := m.Y - r.focused.layout.Content.Y
			if handleTextAreaPointer(r.focused, localX, localY, true) {
				r.MarkDirty()
				consumed = true
			}
		}
	}

	// Left release: stop textarea drag selection.
	if action == input.ActionRelease {
		if r.focused != nil && r.focused.nd != nil && r.focused.nd.Kind == node.TextAreaKind {
			if r.focused.textAreaDragging {
				r.focused.textAreaDragging = false
				if r.focused.nd.TextAreaOpts.OnCopy != nil {
					if text, ok := textAreaSelectedText(r.focused, []rune(r.focused.nd.Text)); ok {
						r.focused.nd.TextAreaOpts.OnCopy(text)
					}
				}
			}
		}
	}

	// Wheel: scroll the hit TextPanel (or nearest ancestor TextPanel).
	if m.Button&(input.MouseWheelUp|input.MouseWheelDown) != 0 {
		for tp := hit; tp != nil; tp = tp.parent {
			if tp.nd != nil && tp.nd.Kind == node.TextPanelKind && !tp.nd.Disabled {
				var press input.KeyPress
				if m.Button&input.MouseWheelUp != 0 {
					press = input.KeyPress{Key: input.KeyUp}
				} else {
					press = input.KeyPress{Key: input.KeyDown}
				}
				if handleTextPanelKey(tp, press) {
					r.MarkDirty()
					consumed = true
				}
				break
			}

			if tp.nd != nil && tp.nd.Kind == node.TextAreaKind && !tp.nd.Disabled {
				delta := 1
				if m.Button&input.MouseWheelUp != 0 {
					delta = -1
				}
				if scrollTextArea(tp, delta) {
					r.MarkDirty()
					consumed = true
				}
				break
			}

			if tp.nd != nil && tp.nd.Kind == node.ScrollableListKind && !tp.nd.Disabled {
				var press input.KeyPress
				if m.Button&input.MouseWheelUp != 0 {
					press = input.KeyPress{Key: input.KeyUp}
				} else {
					press = input.KeyPress{Key: input.KeyDown}
				}
				if handleScrollableListKey(tp, press) {
					r.MarkDirty()
					consumed = true
				}
				break
			}

			if tp == focusRoot {
				break
			}
		}
	}

	// Bubble OnMouse from the hit instance up to the focus root.
	for inst := hit; inst != nil; inst = inst.parent {
		if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnMouse != nil {
			press := input.MousePress{
				X:      m.X,
				Y:      m.Y,
				LocalX: m.X - inst.layout.Content.X,
				LocalY: m.Y - inst.layout.Content.Y,
				Button: m.Button,
				Action: action,
				Mod:    normalizeMod(m.Mod),
			}
			if inst.nd.OnMouse(press) {
				consumed = true
				break
			}
		}
		if inst == focusRoot {
			break
		}
	}

	return consumed
}

// nearestFocusableAncestor walks from hit toward stop, returning the first
// focusable, non-disabled instance. Returns nil if none is found.
func nearestFocusableAncestor(hit, stop *Instance) *Instance {
	for inst := hit; inst != nil; inst = inst.parent {
		if inst.nd != nil && inst.nd.Focusable && !inst.nd.Disabled {
			return inst
		}
		if inst == stop {
			break
		}
	}
	return nil
}

// hitTest returns the deepest instance whose layout rect contains (x, y),
// searching children in reverse order so visually topmost children win.
func hitTest(inst *Instance, x, y int) *Instance {
	for i := len(inst.children) - 1; i >= 0; i-- {
		if hit := hitTest(inst.children[i], x, y); hit != nil {
			return hit
		}
	}
	r := inst.layout.Rect
	if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
		return inst
	}
	return nil
}

// deliverKey walks the instance tree depth-first (children before parent),
// calling the first OnKey handler that consumes the event.
// Used as fallback when nothing is focused.
// Disabled nodes' handlers are skipped but their children are still visited.
func deliverKey(inst *Instance, key event.Key) bool {
	for _, child := range inst.children {
		if deliverKey(child, key) {
			return true
		}
	}
	if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnKey != nil {
		press := input.KeyPress{
			Key:  event.NormalizeKey(key.Key, key.Rune, key.Mod),
			Rune: key.Rune,
			Mod:  normalizeMod(key.Mod),
		}
		return inst.nd.OnKey(press)
	}
	return false
}

func deliverKeyCapture(inst *Instance, key event.Key) bool {
	if inst.nd != nil && !inst.nd.Disabled && inst.nd.OnKeyCapture != nil {
		press := input.KeyPress{
			Key:  event.NormalizeKey(key.Key, key.Rune, key.Mod),
			Rune: key.Rune,
			Mod:  normalizeMod(key.Mod),
		}
		if inst.nd.OnKeyCapture(press) {
			return true
		}
	}
	for _, child := range inst.children {
		if deliverKeyCapture(child, key) {
			return true
		}
	}
	return false
}

func focusedPath(root, focused *Instance) []*Instance {
	if root == nil || focused == nil {
		return nil
	}

	var path []*Instance
	for inst := focused; inst != nil; inst = inst.parent {
		path = append(path, inst)
		if inst == root {
			break
		}
	}
	if len(path) == 0 || path[len(path)-1] != root {
		return nil
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// collectFocusable returns all focusable instances in depth-first order.
func collectFocusable(root *Instance) []*Instance {
	var result []*Instance
	collectFocusableRec(root, &result)
	return result
}

func activeFocusRoot(root *Instance) *Instance {
	if root == nil {
		return nil
	}
	if scoped := findTopmostFocusScope(root); scoped != nil {
		return scoped
	}
	return root
}

func findTopmostFocusScope(inst *Instance) *Instance {
	for i := len(inst.children) - 1; i >= 0; i-- {
		if found := findTopmostFocusScope(inst.children[i]); found != nil {
			return found
		}
	}
	if inst.nd != nil && inst.nd.FocusScope {
		return inst
	}
	return nil
}

func collectFocusableRec(inst *Instance, result *[]*Instance) {
	if inst.nd != nil && inst.nd.Focusable && !inst.nd.Disabled {
		*result = append(*result, inst)
	}
	for _, child := range inst.children {
		collectFocusableRec(child, result)
	}
}

// focusPrev moves focus to the previous focusable node, wrapping around.
func (r *Runtime) focusPrev() {
	if r.root == nil {
		return
	}
	focusable := collectFocusable(activeFocusRoot(r.root))
	if len(focusable) == 0 {
		r.focused = nil
		return
	}
	if r.focused == nil {
		r.focused = focusable[len(focusable)-1]
		return
	}
	for i, inst := range focusable {
		if inst == r.focused {
			r.focused = focusable[(i-1+len(focusable))%len(focusable)]
			return
		}
	}
	// Focused instance no longer in tree; reset to last.
	r.focused = focusable[len(focusable)-1]
}

// focusNext advances focus to the next focusable node, wrapping around.
func (r *Runtime) focusNext() {
	if r.root == nil {
		return
	}
	focusable := collectFocusable(activeFocusRoot(r.root))
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

// ensureFocus is called after each Update. Handles focus-scope transitions,
// stale focused instances, and initial focus assignment.
// Sets dirty if focus changed so the app can re-render to reflect focus state.
func (r *Runtime) ensureFocus() {
	if r.root == nil {
		r.focused = nil
		r.scopeStack = nil
		return
	}

	prev := r.focused
	path := findTopmostFocusScopePath(r.root)

	// Find how many frames in the current stack still match the new scope path.
	common := 0
	for common < len(path) && common < len(r.scopeStack) && r.scopeStack[common].scope == path[common] {
		common++
	}

	// Scopes were removed: pop frames and try to restore focus.
	if common < len(r.scopeStack) {
		restoreTo := r.scopeStack[common].restoreTo
		r.scopeStack = r.scopeStack[:common]
		if canRestoreFocus(r.root, restoreTo) {
			r.focused = restoreTo
		}
	}

	// New scopes were added: push frames.
	for i := common; i < len(path); i++ {
		var restoreTo *Instance
		if i == common {
			restoreTo = r.focused // save current focus before entering new scope
		}
		r.scopeStack = append(r.scopeStack, focusScopeFrame{scope: path[i], restoreTo: restoreTo})
	}

	// Ensure focused is valid within the current active scope.
	focusable := collectFocusable(activeFocusRoot(r.root))
	if r.focused != nil {
		for _, inst := range focusable {
			if inst == r.focused {
				if r.focused != prev {
					r.dirty = true
				}
				return // still valid
			}
		}
		// Previously focused instance is gone or now outside active scope.
	}

	if af := firstAutoFocus(focusable); af != nil {
		r.focused = af
	} else if len(focusable) > 0 {
		r.focused = focusable[0]
	} else {
		r.focused = nil
	}
	if r.focused != prev {
		r.dirty = true
	}
}

// canRestoreFocus reports whether target is a valid, focusable instance
// within the current active focus root.
func canRestoreFocus(root *Instance, target *Instance) bool {
	if target == nil {
		return false
	}
	for _, inst := range collectFocusable(activeFocusRoot(root)) {
		if inst == target {
			return true
		}
	}
	return false
}

// findTopmostFocusScopePath returns the chain of focus scopes from the outermost
// scope that contains the topmost scope, down to the topmost scope itself.
// Returns nil if no focus scope is present in the tree.
func findTopmostFocusScopePath(inst *Instance) []*Instance {
	if inst == nil {
		return nil
	}
	for i := len(inst.children) - 1; i >= 0; i-- {
		if path := findTopmostFocusScopePath(inst.children[i]); len(path) > 0 {
			if inst.nd != nil && inst.nd.FocusScope {
				return append([]*Instance{inst}, path...)
			}
			return path
		}
	}
	if inst.nd != nil && inst.nd.FocusScope {
		return []*Instance{inst}
	}
	return nil
}

// firstAutoFocus returns the first focusable instance with AutoFocus set.
func firstAutoFocus(focusable []*Instance) *Instance {
	for _, inst := range focusable {
		if inst.nd != nil && inst.nd.AutoFocus {
			return inst
		}
	}
	return nil
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

	if n.Kind == node.TextPanelKind {
		resetTextPanelState(inst)
	}
	if n.Kind == node.TextAreaKind {
		resetTextAreaState(inst)
	}
	if n.Kind == node.ScrollableListKind {
		resetScrollableListState(inst)
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
		unmount(inst)
		return mount(rt, parent, n)
	}

	inst.nd = n
	inst.kind = n.Kind
	inst.key = n.Key

	if n.Kind == node.TextPanelKind {
		applyTextPanelReset(inst)
	}

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
	reused := make(map[*Instance]bool, len(old))

	for i, nc := range newChildren {
		var matched *Instance

		if nc.Key != "" {
			// Try keyed match.
			if o, ok := keyedOld[nc.Key]; ok && sameKind(o, nc) {
				matched = o
				delete(keyedOld, nc.Key)
			}
		} else {
			// Try positional match (same kind, no key).
			if i < len(old) && !usedOld[i] && old[i].key == "" && sameKind(old[i], nc) {
				matched = old[i]
				usedOld[i] = true
			}
		}

		if matched != nil {
			reused[matched] = true
			next[i] = reconcile(rt, inst, matched, nc)
		} else {
			next[i] = mount(rt, inst, nc)
		}
	}

	for _, oldInst := range old {
		if !reused[oldInst] {
			unmount(oldInst)
		}
	}

	inst.children = next
}

// sameKind reports whether an instance and new node have the same identity.
func sameKind(inst *Instance, n *node.Node) bool {
	if inst.kind != n.Kind {
		return false
	}
	if inst.key != n.Key {
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
	child := n.CompFn()
	if inst.hookIdx < len(inst.hookSlots) {
		panic("tui: hook order changed: fewer hooks called than previous render")
	}
	return child
}

func isInstanceMounted(root, target *Instance) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, child := range root.children {
		if isInstanceMounted(child, target) {
			return true
		}
	}
	return false
}

func unmount(inst *Instance) {
	if inst == nil {
		return
	}

	for _, child := range inst.children {
		unmount(child)
	}

	for i := range inst.hookSlots {
		slot := &inst.hookSlots[i]
		if slot.kind == hookEffect && slot.effect.cleanup != nil {
			slot.effect.cleanup()
			slot.effect.cleanup = nil
		}
	}

	inst.children = nil
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
func renderInstance(inst *Instance, buf *screen.Buffer, inherited style.Style, cursor *cursorState) {
	r := inst.layout.Rect
	content := inst.layout.Content

	if inst.nd.Kind == node.OverlayKind {
		s := style.MergeVisual(inherited, inst.nd.Style)
		fillStyle := screenCellStyleFromStyle(s)

		if inst.nd.Style.Background.IsSpecified() {
			buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		}
		drawBorders(buf, r, s.Border, fillStyle)

		for _, child := range inst.children {
			renderInstance(child, buf, s, cursor)
		}
		return
	}

	if inst.nd.Kind == node.TextPanelKind {
		s := style.MergeVisual(inherited, inst.nd.Style)
		fillStyle := screenCellStyleFromStyle(s)
		buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		drawBorders(buf, r, s.Border, fillStyle)

		renderTextPanel(inst, buf, content, s)
		return
	}

	if inst.nd.Kind == node.TextAreaKind {
		s := style.MergeVisual(inherited, inst.nd.Style)
		fillStyle := screenCellStyleFromStyle(s)
		buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		drawBorders(buf, r, s.Border, fillStyle)

		renderTextArea(inst, buf, content, s, cursor)
		return
	}

	if inst.nd.Kind == node.ScrollableListKind {
		s := style.MergeVisual(inherited, inst.nd.Style)
		if inst.runtime != nil && inst.runtime.focused == inst {
			s = overlayRuntimeVisualStyle(s, inst.nd.ScrollableListOpts.FocusedStyle)
		}
		fillStyle := screenCellStyleFromStyle(s)
		buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		drawBorders(buf, r, s.Border, fillStyle)

		renderScrollableList(inst, buf, content, s)
		return
	}

	if inst.nd.Kind == node.ViewKind {
		s := style.MergeVisual(inherited, inst.nd.Style)
		fillStyle := screenCellStyleFromStyle(s)
		buf.FillRect(r.X, r.Y, r.W, r.H, ' ', fillStyle)
		drawBorders(buf, r, s.Border, fillStyle)

		if inst.nd.CursorVisible && cursor != nil && content.W > 0 && content.H > 0 {
			cx := inst.nd.CursorX
			if cx < 0 {
				cx = 0
			}
			if cx >= content.W {
				cx = content.W - 1
			}
			cy := inst.nd.CursorY
			if cy < 0 {
				cy = 0
			}
			if cy >= content.H {
				cy = content.H - 1
			}
			cursor.visible = true
			cursor.x = content.X + cx
			cursor.y = content.Y + cy
		}

		for _, child := range inst.children {
			renderInstance(child, buf, s, cursor)
		}
		return
	}

	if inst.nd.Kind == node.TextKind {
		opts := inst.nd.TextOpts
		s := style.MergeVisual(inherited, opts.Style)
		textStyle := screenCellStyleFromStyle(s)
		drawMultilineText(buf, content, inst.nd.Text, textStyle, opts.Align)
		return
	}

	if inst.nd.Kind == node.RichTextKind {
		opts := inst.nd.TextOpts
		baseStyle := style.MergeVisual(inherited, opts.Style)
		drawRichText(buf, content, inst.nd.Spans, baseStyle, opts.Align)
		return
	}

	// Component or Keyed: render children.
	for _, child := range inst.children {
		renderInstance(child, buf, inherited, cursor)
	}
}

func screenCellStyleFromStyle(s style.Style) screen.CellStyle {
	return screen.CellStyle{
		Fg:            s.Foreground,
		Bg:            s.Background,
		Bold:          s.Bold,
		Italic:        s.Italic,
		Underline:     s.Underline,
		Faint:         s.Faint,
		Strikethrough: s.Strikethrough,
		Reverse:       s.Reverse,
	}
}

// alignedXByWidth computes the starting x for a single line given alignment.
func alignedXByWidth(rect style.Rect, width int, align node.TextAlign) int {
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

// alignedX computes the starting x for a single line of text given alignment.
func alignedX(rect style.Rect, line string, align node.TextAlign) int {
	return alignedXByWidth(rect, runewidth.StringWidth(line), align)
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

func drawRichText(buf *screen.Buffer, rect style.Rect, spans []node.TextSpan, base style.Style, align node.TextAlign) {
	lines := node.SplitTextSpansLines(spans)
	for i, line := range lines {
		if i >= rect.H {
			break
		}
		logicalX := alignedXByWidth(rect, node.RichTextLineWidth(line), align)
		y := rect.Y + i
		for _, span := range line {
			spanStyle := screenCellStyleFromStyle(style.MergeVisual(base, span.Style))
			buf.DrawTextClipped(logicalX, y, span.Text, spanStyle, rect.X, rect.Y, rect.W, rect.H)
			logicalX += runewidth.StringWidth(span.Text)
		}
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
