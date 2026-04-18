package runtime

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/style"
)

// helper: make a view node.
func viewND(children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, Children: children}
}

// helper: make a text node.
func textND(t string) *node.Node {
	return &node.Node{Kind: node.TextKind, Text: t}
}

// helper: make a keyed node.
func keyedND(key string, child *node.Node) *node.Node {
	return &node.Node{Kind: node.KeyedKind, Key: key, Children: []*node.Node{child}}
}

func TestMountView(t *testing.T) {
	rt := New()
	n := viewND(textND("hello"), textND("world"))
	rt.Update(n)
	if rt.root == nil {
		t.Fatal("root is nil")
	}
	if len(rt.root.children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(rt.root.children))
	}
	if rt.root.children[0].nd.Text != "hello" {
		t.Errorf("child 0 text: %q", rt.root.children[0].nd.Text)
	}
}

func TestReconcilePreservesInstance(t *testing.T) {
	rt := New()

	n1 := viewND(textND("a"))
	rt.Update(n1)
	child1 := rt.root.children[0]

	n2 := viewND(textND("b"))
	rt.Update(n2)
	child2 := rt.root.children[0]

	// Same position, same kind: instance should be reused.
	if child1 != child2 {
		t.Error("expected same instance to be reused")
	}
	if child2.nd.Text != "b" {
		t.Errorf("expected updated text 'b', got %q", child2.nd.Text)
	}
}

func TestReconcileRemountsOnKindChange(t *testing.T) {
	rt := New()

	n1 := viewND(viewND())
	rt.Update(n1)
	child1 := rt.root.children[0]

	// Replace view child with text child (different kind).
	n2 := viewND(textND("new"))
	rt.Update(n2)
	child2 := rt.root.children[0]

	if child1 == child2 {
		t.Error("expected different instance on kind change")
	}
}

func TestKeyedReconciliation(t *testing.T) {
	rt := New()

	// Mount with two keyed children: A, B.
	n1 := viewND(
		keyedND("A", textND("a")),
		keyedND("B", textND("b")),
	)
	rt.Update(n1)
	instA1 := rt.root.children[0]
	instB1 := rt.root.children[1]

	// Update reversing order: B, A.
	n2 := viewND(
		keyedND("B", textND("b2")),
		keyedND("A", textND("a2")),
	)
	rt.Update(n2)
	instB2 := rt.root.children[0]
	instA2 := rt.root.children[1]

	// Keyed instances should match by key, not position.
	if instA1 != instA2 {
		t.Error("expected keyed instance A to be preserved")
	}
	if instB1 != instB2 {
		t.Error("expected keyed instance B to be preserved")
	}
}

func TestComponentRendering(t *testing.T) {
	rendered := 0
	compFn := func() *node.Node {
		rendered++
		return textND("from component")
	}
	compID := uintptr(1234) // fake ID

	rt := New()
	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: compID}
	rt.Update(n)

	if rendered != 1 {
		t.Errorf("component should have been rendered once, got %d", rendered)
	}
	if len(rt.root.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(rt.root.children))
	}
	if rt.root.children[0].nd.Text != "from component" {
		t.Errorf("unexpected child text: %q", rt.root.children[0].nd.Text)
	}
}

func TestUseStatePreserved(t *testing.T) {
	count := 0
	var setCount func(int)

	compFn := func() *node.Node {
		c, s := UseState(0)
		count = c
		setCount = s
		return textND("count")
	}
	compID := uintptr(9999)

	rt := New()
	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: compID}
	rt.Update(n)

	if count != 0 {
		t.Errorf("initial count: %d", count)
	}

	// Update state.
	setCount(42)
	rt.Update(n) // re-render

	if count != 42 {
		t.Errorf("after setState count: %d, want 42", count)
	}
}

func TestUseStatePanicsOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic calling UseState outside render")
		}
	}()
	UseState(0)
}

func TestConditionalHookCausesTypeMismatchPanic(t *testing.T) {
	rt := New()
	condition := true
	compFn := func() *node.Node {
		if condition {
			UseState(0)
		} else {
			UseState("string")
		}
		return textND("foo")
	}

	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: 1}
	rt.Update(n)

	condition = false
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on type mismatch for conditional hook")
		}
	}()
	rt.Update(n)
}

func focusableND(children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, Focusable: true, Children: children}
}

func TestInstanceHasStableID(t *testing.T) {
	rt := New()
	rt.Update(viewND(textND("a")))
	id1 := rt.root.id
	if id1 == 0 {
		t.Fatal("expected non-zero ID")
	}
	rt.Update(viewND(textND("b")))
	if rt.root.id != id1 {
		t.Error("reconciled instance should keep its ID")
	}
}

func TestRemountGetsNewID(t *testing.T) {
	rt := New()
	rt.Update(viewND(textND("a")))
	id1 := rt.root.id

	rt.Update(textND("b")) // kind change forces remount
	id2 := rt.root.id
	if id2 == id1 {
		t.Error("remounted instance should get a new ID")
	}
}

func TestParentPointers(t *testing.T) {
	rt := New()
	rt.Update(viewND(viewND(textND("leaf"))))
	root := rt.root
	child := root.children[0]
	leaf := child.children[0]

	if child.parent != root {
		t.Error("child.parent should point to root")
	}
	if leaf.parent != child {
		t.Error("leaf.parent should point to child")
	}
	if root.parent != nil {
		t.Error("root.parent should be nil")
	}
}

func TestFocusFirstFocusableOnMount(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	if rt.focused == nil {
		t.Fatal("expected focus to be set after mount")
	}
	if rt.focused != rt.root.children[0] {
		t.Error("expected first focusable child to be focused")
	}
}

func TestFocusNextCycles(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	first := rt.focused

	rt.focusNext()
	second := rt.focused
	if second == first {
		t.Error("focusNext should move to second focusable")
	}

	rt.focusNext()
	if rt.focused != first {
		t.Error("focusNext should wrap back to first focusable")
	}
}

func TestFocusSurvivesReconcile(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND()))
	focused := rt.focused

	rt.Update(viewND(focusableND(textND("new child"))))
	if rt.focused != focused {
		t.Error("focus should survive reconcile of the same focusable instance")
	}
}

func TestFocusMovesWhenFocusedNodeRemoved(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	rt.focused = rt.root.children[0] // focus first

	// Remove both focusable children.
	rt.Update(viewND())
	if rt.focused != nil {
		t.Error("focus should be nil when all focusable nodes removed")
	}
}

func TestKeyDeliveredToFocused(t *testing.T) {
	rt := New()
	received := false
	handler := func(kp node.KeyPress) bool {
		if kp.Rune == 'x' {
			received = true
			return true
		}
		return false
	}
	rt.Update(viewND(
		&node.Node{Kind: node.ViewKind, Focusable: true, OnKey: handler},
		&node.Node{Kind: node.ViewKind, Focusable: true, OnKey: func(node.KeyPress) bool {
			t.Error("unfocused node should not receive key")
			return true
		}},
	))

	// Focus is on the first focusable node.
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'x'}})
	if !received {
		t.Error("focused node should have received the key")
	}
}

func TestKeyBubblesUpParentChain(t *testing.T) {
	rt := New()
	childHandled := false
	parentHandled := false

	child := &node.Node{
		Kind:      node.ViewKind,
		Focusable: true,
		OnKey: func(kp node.KeyPress) bool {
			childHandled = true
			return false // not consumed; let it bubble
		},
	}
	parent := &node.Node{
		Kind:     node.ViewKind,
		Children: []*node.Node{child},
		OnKey: func(kp node.KeyPress) bool {
			parentHandled = true
			return true
		},
	}
	rt.Update(parent)

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'z'}})
	if !childHandled {
		t.Error("focused child should have been tried first")
	}
	if !parentHandled {
		t.Error("event should have bubbled to parent")
	}
}

func TestMarkDirty(t *testing.T) {
	rt := New()
	if rt.IsDirty() {
		// Initial dirty is expected.
	}
	rt.Update(viewND())
	if rt.IsDirty() {
		t.Error("after Update, should not be dirty")
	}
	rt.MarkDirty()
	if !rt.IsDirty() {
		t.Error("after MarkDirty, should be dirty")
	}
}

func TestRunLayout(t *testing.T) {
	rt := New()
	n := viewND(
		&node.Node{Kind: node.ViewKind, Style: style.Style{Width: style.Cells(20)}},
	)
	rt.Update(n)
	rt.RunLayout(80, 24)

	if rt.root.layout.Rect.W != 80 {
		t.Errorf("root width: %d", rt.root.layout.Rect.W)
	}
	if len(rt.root.children) > 0 {
		child := rt.root.children[0]
		if child.layout.Rect.W != 20 {
			t.Errorf("child width: %d, want 20", child.layout.Rect.W)
		}
	}
}

func TestUseFocusedWithDirectFocusableChild(t *testing.T) {
	focusedValues := []bool{}
	compFn := func() *node.Node {
		focusedValues = append(focusedValues, IsFocused())
		return focusableND()
	}
	compID := uintptr(5555)

	rt := New()
	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: compID}
	rt.Update(n)
	// After initial mount, focus is set and may mark dirty. Re-render to get updated focused state.
	if rt.IsDirty() {
		rt.Update(n)
	}

	// Should have rendered at least twice if focus was set, or once if nothing is focusable
	if len(focusedValues) == 0 {
		t.Fatal("component should have been rendered")
	}
	// Last render should show focused=true since the component contains the focused instance
	if !focusedValues[len(focusedValues)-1] {
		t.Errorf("UseFocused should be true for focused component's subtree. Values: %v", focusedValues)
	}
}

func TestUseFocusedThroughNonFocusableWrapper(t *testing.T) {
	focusedValues := []bool{}
	compFn := func() *node.Node {
		focusedValues = append(focusedValues, IsFocused())
		// Return a non-focusable wrapper containing the focusable node
		return viewND(focusableND())
	}
	compID := uintptr(6666)

	rt := New()
	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: compID}
	rt.Update(n)
	if rt.IsDirty() {
		rt.Update(n)
	}

	if len(focusedValues) == 0 {
		t.Fatal("component should have been rendered")
	}
	if !focusedValues[len(focusedValues)-1] {
		t.Errorf("UseFocused should be true when focused node is nested through non-focusable wrapper. Values: %v", focusedValues)
	}
}

func TestUseFocusedThroughKeyedChild(t *testing.T) {
	focusedValues := []bool{}
	compFn := func() *node.Node {
		focusedValues = append(focusedValues, IsFocused())
		return keyedND("wrapper", focusableND())
	}
	compID := uintptr(7777)

	rt := New()
	n := &node.Node{Kind: node.ComponentKind, CompFn: compFn, CompID: compID}
	rt.Update(n)
	if rt.IsDirty() {
		rt.Update(n)
	}

	if len(focusedValues) == 0 {
		t.Fatal("component should have been rendered")
	}
	if !focusedValues[len(focusedValues)-1] {
		t.Errorf("UseFocused should be true when focused node is nested through keyed child. Values: %v", focusedValues)
	}
}

func TestUseFocusedFalseForSiblingComponent(t *testing.T) {
	focusedValues1 := []bool{}
	focusedValues2 := []bool{}

	compFn1 := func() *node.Node {
		focusedValues1 = append(focusedValues1, IsFocused())
		return focusableND()
	}
	compID1 := uintptr(8888)

	compFn2 := func() *node.Node {
		focusedValues2 = append(focusedValues2, IsFocused())
		return viewND() // non-focusable
	}
	compID2 := uintptr(9999)

	rt := New()
	n := viewND(
		&node.Node{Kind: node.ComponentKind, CompFn: compFn1, CompID: compID1},
		&node.Node{Kind: node.ComponentKind, CompFn: compFn2, CompID: compID2},
	)
	rt.Update(n)
	if rt.IsDirty() {
		rt.Update(n)
	}

	if len(focusedValues1) == 0 || len(focusedValues2) == 0 {
		t.Fatal("both components should have been rendered")
	}
	if !focusedValues1[len(focusedValues1)-1] {
		t.Errorf("first component should have focused true. Values: %v", focusedValues1)
	}
	if focusedValues2[len(focusedValues2)-1] {
		t.Errorf("second sibling component should have focused false. Values: %v", focusedValues2)
	}
}

func autoFocusND(children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, Focusable: true, AutoFocus: true, Children: children}
}

func disabledFocusableND(children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, Focusable: true, Disabled: true, Children: children}
}

func TestFocusPrevCyclesBackward(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND(), focusableND()))
	rt.focused = rt.root.children[0]

	rt.focusPrev()
	// From first, wraps to last.
	want := rt.root.children[2]
	if rt.focused != want {
		t.Error("focusPrev from first should wrap to last")
	}
}

func TestFocusPrevMovesBackward(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND(), focusableND()))
	rt.focused = rt.root.children[2]

	rt.focusPrev()
	want := rt.root.children[1]
	if rt.focused != want {
		t.Error("focusPrev from last should move to second")
	}
}

func TestShiftTabCallsFocusPrev(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	rt.focused = rt.root.children[1] // start on second

	rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyBacktab},
	})
	if rt.focused != rt.root.children[0] {
		t.Error("Shift+Tab should move focus to first node")
	}
}

func TestPlainTabStillMovesForward(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	rt.focused = rt.root.children[0]

	rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyTab},
	})
	if rt.focused != rt.root.children[1] {
		t.Error("Tab should move focus forward")
	}
}

func TestAutoFocusWinsOnInitialMount(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), autoFocusND()))
	if rt.focused == nil {
		t.Fatal("expected focus to be set")
	}
	if rt.focused != rt.root.children[1] {
		t.Error("AutoFocus node should be focused on initial mount")
	}
}

func TestAutoFocusDoesNotStealExistingFocus(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	first := rt.root.children[0]
	rt.focused = first

	// Update: second child now has AutoFocus. Existing focus should stay.
	rt.Update(viewND(focusableND(), autoFocusND()))
	if rt.focused != first {
		t.Error("AutoFocus should not steal focus from an already-focused node")
	}
}

func TestDisabledNodesAreSkippedInFocusTraversal(t *testing.T) {
	rt := New()
	rt.Update(viewND(disabledFocusableND(), focusableND()))
	if rt.focused == nil {
		t.Fatal("expected focus to be set")
	}
	if rt.focused != rt.root.children[1] {
		t.Error("disabled node should be skipped; second child should be focused")
	}
}

func TestFocusedDisabledNodeLosesFocusOnUpdate(t *testing.T) {
	rt := New()
	rt.Update(viewND(focusableND(), focusableND()))
	rt.focused = rt.root.children[0]

	// Update: first child becomes disabled.
	rt.Update(viewND(disabledFocusableND(), focusableND()))
	if rt.focused == rt.root.children[0] {
		t.Error("focus should leave disabled node after update")
	}
	if rt.focused != rt.root.children[1] {
		t.Error("focus should move to second (non-disabled) node")
	}
}

func TestDisabledParentDoesNotDisableChild(t *testing.T) {
	rt := New()
	// Disabled outer view wrapping a focusable child.
	outer := &node.Node{
		Kind:     node.ViewKind,
		Disabled: true, // disabled but not focusable
		Children: []*node.Node{focusableND()},
	}
	rt.Update(outer)
	if rt.focused == nil {
		t.Fatal("child of disabled non-focusable parent should still be focusable")
	}
	if rt.focused != rt.root.children[0] {
		t.Error("expected child instance to be focused")
	}
}

func TestDisabledParentOnKeySkippedDuringBubble(t *testing.T) {
	rt := New()
	parentHandled := false
	childHandled := false

	child := &node.Node{
		Kind:      node.ViewKind,
		Focusable: true,
		OnKey: func(kp node.KeyPress) bool {
			childHandled = true
			return false // not consumed; let it bubble
		},
	}
	parent := &node.Node{
		Kind:     node.ViewKind,
		Disabled: true, // disabled parent
		Children: []*node.Node{child},
		OnKey: func(kp node.KeyPress) bool {
			parentHandled = true
			return true
		},
	}
	rt.Update(parent)

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'x'}})
	if !childHandled {
		t.Error("focused child should have received the key")
	}
	if parentHandled {
		t.Error("disabled parent's OnKey should not be called during bubbling")
	}
}

func TestFallbackDeliverySkipsDisabledHandler(t *testing.T) {
	rt := New()
	disabledHandled := false
	childHandled := false

	child := &node.Node{
		Kind:  node.ViewKind,
		OnKey: func(kp node.KeyPress) bool { childHandled = true; return true },
	}
	disabled := &node.Node{
		Kind:     node.ViewKind,
		Disabled: true,
		Children: []*node.Node{child},
		OnKey:    func(kp node.KeyPress) bool { disabledHandled = true; return true },
	}
	rt.Update(disabled)

	// Clear focus to trigger fallback delivery.
	rt.focused = nil
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'y'}})
	if disabledHandled {
		t.Error("disabled node's handler should not be called in fallback delivery")
	}
	if !childHandled {
		t.Error("child of disabled parent should still be visited in fallback delivery")
	}
}

func TestFallbackDeliveryStillVisitsChildOfDisabledParent(t *testing.T) {
	rt := New()
	childHandled := false

	child := &node.Node{
		Kind:  node.ViewKind,
		OnKey: func(kp node.KeyPress) bool { childHandled = true; return true },
	}
	disabled := &node.Node{
		Kind:     node.ViewKind,
		Disabled: true,
		Children: []*node.Node{child},
	}
	rt.Update(disabled)

	// Clear focus to trigger fallback delivery.
	rt.focused = nil
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'z'}})
	if !childHandled {
		t.Error("child of disabled parent should be visited even though parent is disabled")
	}
}

func TestNormalizeModNone(t *testing.T) {
	got := normalizeMod(0)
	if got != 0 {
		t.Errorf("normalizeMod(0) = %v, want 0", got)
	}
}

func TestNormalizeModCtrl(t *testing.T) {
	got := normalizeMod(tcell.ModCtrl)
	if got != input.ModCtrl {
		t.Errorf("normalizeMod(ModCtrl) = %v, want ModCtrl", got)
	}
}

func TestNormalizeModAlt(t *testing.T) {
	got := normalizeMod(tcell.ModAlt)
	if got != input.ModAlt {
		t.Errorf("normalizeMod(ModAlt) = %v, want ModAlt", got)
	}
}

func TestNormalizeModShift(t *testing.T) {
	got := normalizeMod(tcell.ModShift)
	if got != input.ModShift {
		t.Errorf("normalizeMod(ModShift) = %v, want ModShift", got)
	}
}

func TestNormalizeModCtrlAlt(t *testing.T) {
	got := normalizeMod(tcell.ModCtrl | tcell.ModAlt)
	expected := input.ModCtrl | input.ModAlt
	if got != expected {
		t.Errorf("normalizeMod(ModCtrl|ModAlt) = %v, want %v", got, expected)
	}
}

func TestNormalizeModCtrlShift(t *testing.T) {
	got := normalizeMod(tcell.ModCtrl | tcell.ModShift)
	expected := input.ModCtrl | input.ModShift
	if got != expected {
		t.Errorf("normalizeMod(ModCtrl|ModShift) = %v, want %v", got, expected)
	}
}

func TestNormalizeModAltShift(t *testing.T) {
	got := normalizeMod(tcell.ModAlt | tcell.ModShift)
	expected := input.ModAlt | input.ModShift
	if got != expected {
		t.Errorf("normalizeMod(ModAlt|ModShift) = %v, want %v", got, expected)
	}
}

func TestNormalizeModAll(t *testing.T) {
	got := normalizeMod(tcell.ModCtrl | tcell.ModAlt | tcell.ModShift)
	expected := input.ModCtrl | input.ModAlt | input.ModShift
	if got != expected {
		t.Errorf("normalizeMod(ModCtrl|ModAlt|ModShift) = %v, want %v", got, expected)
	}
}

func focusScopeND(children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, FocusScope: true, Children: children}
}

func TestFocusScopeTrapsTabTraversal(t *testing.T) {
	rt := New()
	background := focusableND()
	f1 := focusableND()
	f2 := focusableND()
	scope := focusScopeND(f1, f2)
	root := viewND(background, scope)
	rt.Update(root)

	// Initial focus should be in the focus scope because it's the topmost scope
	// and we search for focusable nodes within it.
	// The active focus root is the scope, so background is excluded.
	if rt.focused != rt.root.children[1].children[0] {
		t.Fatalf("expected focus to be first node in scope")
	}

	rt.focusNext()
	if rt.focused != rt.root.children[1].children[1] {
		t.Error("focusNext should move to second node in scope")
	}

	rt.focusNext()
	if rt.focused != rt.root.children[1].children[0] {
		t.Error("focusNext should wrap back to first node in scope, skipping background")
	}
}

func TestFocusScopeMovesFocusInsideWhenOpened(t *testing.T) {
	rt := New()
	background := focusableND()
	rt.Update(background)
	if rt.focused == nil {
		t.Fatal("background should be focused")
	}

	// Update to add a focus scope.
	f1 := focusableND()
	scope := focusScopeND(f1)
	root := viewND(background, scope)
	rt.Update(root)

	if rt.focused != rt.root.children[1].children[0] {
		t.Error("focus should have moved into the new focus scope")
	}
}

func TestFocusScopePreventsFallbackDeliveryToBackground(t *testing.T) {
	rt := New()
	backgroundHandled := false
	background := &node.Node{
		Kind:  node.ViewKind,
		OnKey: func(kp node.KeyPress) bool { backgroundHandled = true; return true },
	}
	scopeHandled := false
	scopeChild := &node.Node{
		Kind:  node.ViewKind,
		OnKey: func(kp node.KeyPress) bool { scopeHandled = true; return true },
	}
	scope := &node.Node{
		Kind:       node.ViewKind,
		FocusScope: true,
		Children:   []*node.Node{scopeChild},
	}
	root := viewND(background, scope)
	rt.Update(root)
	rt.focused = nil // ensure fallback delivery

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Rune: 'x'}})
	if backgroundHandled {
		t.Error("background should not have received key when focus scope is active")
	}
	if !scopeHandled {
		t.Error("node inside focus scope should have received key")
	}
}

func TestTopmostFocusScopeWins(t *testing.T) {
	rt := New()
	f1 := focusableND()
	scope1 := focusScopeND(f1)

	f2 := focusableND()
	scope2 := focusScopeND(f2)

	// Use Overlay to have two overlapping scopes.
	// Later child (scope2) is topmost.
	root := &node.Node{
		Kind:     node.OverlayKind,
		Children: []*node.Node{scope1, scope2},
	}
	rt.Update(root)

	if rt.focused != rt.root.children[1].children[0] { // f2
		t.Errorf("expected focus in topmost scope (scope2), got %v", rt.focused)
	}
}

func TestDeepestFocusScopeWinsEvenIfSiblingIsFocused(t *testing.T) {
	rt := New()

	f1 := focusableND()
	scope1 := focusScopeND(f1)

	f2 := focusableND()
	scope2 := focusScopeND(f2)

	// scope2 comes after scope1, so it wins findTopmostFocusScope.
	root := viewND(scope1, scope2)
	rt.Update(root)

	// Initially focus should be in scope2 because it is later in the tree.
	if rt.focused != rt.root.children[1].children[0] {
		t.Errorf("expected focus in scope2, got %v", rt.focused)
	}

	// Manually focus f1 in scope1.
	rt.focused = rt.root.children[0].children[0]

	// Now if we tab, it should wrap within scope2, because scope2 is the active focus root.
	// f1 is not in collectFocusable(scope2), so focusNext resets it to focusable[0] of scope2.
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyTab}})
	if rt.focused != rt.root.children[1].children[0] {
		t.Errorf("expected focus to jump to scope2 on tab, got %v", rt.focused)
	}
}

func TestDuplicateKeysDoNotReuseSameInstanceTwice(t *testing.T) {
	rt := New()

	// Initial render with one keyed child.
	node1 := &node.Node{
		Kind: node.ViewKind,
		Children: []*node.Node{
			{Kind: node.ViewKind, Key: "a"},
		},
	}
	rt.Update(node1)
	instA := rt.root.children[0]

	// Update with two children sharing the same key "a".
	// The runtime should NOT reuse instA for both.
	node2 := &node.Node{
		Kind: node.ViewKind,
		Children: []*node.Node{
			{Kind: node.ViewKind, Key: "a"},
			{Kind: node.ViewKind, Key: "a"},
		},
	}
	rt.Update(node2)

	first := rt.root.children[0]
	second := rt.root.children[1]

	if first != instA {
		t.Errorf("first child should be instA, got %p", first)
	}
	if second == instA {
		t.Errorf("second child should NOT be instA (duplicate key)")
	}
}

func TestFocusScopeRestoresPreviousFocusWhenClosed(t *testing.T) {
	rt := New()

	first := focusableND()
	opener := focusableND()

	rt.Update(viewND(first, opener))
	rt.focused = rt.root.children[1] // opener

	scopeChild := focusableND()
	rt.Update(viewND(first, opener, focusScopeND(scopeChild)))

	if rt.focused != rt.root.children[2].children[0] {
		t.Fatal("focus should move into opened focus scope")
	}

	rt.Update(viewND(first, opener))

	if rt.focused != rt.root.children[1] {
		t.Fatal("focus should restore to previously focused opener")
	}
}

func TestFocusScopeFallsBackWhenPreviousFocusRemoved(t *testing.T) {
	rt := New()

	first := focusableND()
	opener := focusableND()

	rt.Update(viewND(first, opener))
	rt.focused = rt.root.children[1]

	scopeChild := focusableND()
	rt.Update(viewND(first, opener, focusScopeND(scopeChild)))

	if rt.focused != rt.root.children[2].children[0] {
		t.Fatal("focus should move into opened focus scope")
	}

	// Remove opener while closing the scope.
	rt.Update(viewND(first))

	if rt.focused != rt.root.children[0] {
		t.Fatal("focus should fall back to first focusable when previous focus is gone")
	}
}

func TestFocusScopeFallsBackWhenPreviousFocusDisabled(t *testing.T) {
	rt := New()

	first := focusableND()
	opener := focusableND()

	rt.Update(viewND(first, opener))
	rt.focused = rt.root.children[1]

	scopeChild := focusableND()
	rt.Update(viewND(first, opener, focusScopeND(scopeChild)))

	if rt.focused != rt.root.children[2].children[0] {
		t.Fatal("focus should move into opened focus scope")
	}

	// Same position/kind, but opener is now disabled.
	rt.Update(viewND(first, disabledFocusableND()))

	if rt.focused != rt.root.children[0] {
		t.Fatal("focus should fall back when previous focus is disabled")
	}
}

// --- Ticket 6: Nested focus scope tests ---

func TestNestedFocusScopeRestoresOuterScopeOnInnerClose(t *testing.T) {
	rt := New()

	bg := focusableND()
	outerOpener := focusableND()
	outerOther := focusableND()
	innerChild := focusableND()

	// Setup: bg + outer scope (outerOpener + outerOther) + inner scope (innerChild)
	rt.Update(viewND(bg, focusScopeND(outerOpener, outerOther, focusScopeND(innerChild))))

	// Focus should land in the inner scope (deepest/topmost).
	if rt.focused != rt.root.children[1].children[2].children[0] {
		t.Fatal("focus should be in inner scope initially")
	}

	// Close inner scope; outer scope remains.
	rt.Update(viewND(bg, focusScopeND(outerOpener, outerOther)))

	// Focus should restore to something inside the outer scope (not bg).
	outerScope := rt.root.children[1]
	innerFocusable := collectFocusable(outerScope)
	found := false
	for _, inst := range innerFocusable {
		if inst == rt.focused {
			found = true
			break
		}
	}
	if !found {
		t.Error("focus should be inside the outer scope after closing inner scope")
	}
	if rt.focused == rt.root.children[0] {
		t.Error("focus must not escape to background when outer scope is still active")
	}
}

func TestNestedFocusScopeRestoresToBackgroundWhenBothClose(t *testing.T) {
	rt := New()

	bg := focusableND()
	outerChild := focusableND()
	innerChild := focusableND()

	rt.Update(viewND(bg))
	rt.focused = rt.root.children[0] // bg focused

	// Open outer scope.
	rt.Update(viewND(bg, focusScopeND(outerChild)))
	if rt.focused != rt.root.children[1].children[0] {
		t.Fatal("focus should move into outer scope")
	}

	// Open inner scope inside outer.
	rt.Update(viewND(bg, focusScopeND(outerChild, focusScopeND(innerChild))))
	if rt.focused != rt.root.children[1].children[1].children[0] {
		t.Fatal("focus should move into inner scope")
	}

	// Close both scopes.
	rt.Update(viewND(bg))

	if rt.focused != rt.root.children[0] {
		t.Fatal("focus should restore to background when both scopes close")
	}
}

func TestNestedFocusScopeFallsBackWhenRestoreTargetRemoved(t *testing.T) {
	rt := New()

	bg := focusableND()
	outerOpener := focusableND()
	innerChild := focusableND()

	rt.Update(viewND(bg, outerOpener))
	rt.focused = rt.root.children[1] // outerOpener

	// Open outer scope.
	rt.Update(viewND(bg, outerOpener, focusScopeND(innerChild)))
	if rt.focused != rt.root.children[2].children[0] {
		t.Fatal("focus should move into scope")
	}

	// Close scope, remove outerOpener simultaneously.
	rt.Update(viewND(bg))

	// outerOpener is gone, focus should fall back to bg.
	if rt.focused != rt.root.children[0] {
		t.Fatal("focus should fall back to bg when restore target removed")
	}
}

func TestNestedFocusScopeFallsBackWhenRestoreTargetDisabled(t *testing.T) {
	rt := New()

	bg := focusableND()
	opener := focusableND()
	scopeChild := focusableND()

	rt.Update(viewND(bg, opener))
	rt.focused = rt.root.children[1]

	rt.Update(viewND(bg, opener, focusScopeND(scopeChild)))
	if rt.focused != rt.root.children[2].children[0] {
		t.Fatal("focus should move into scope")
	}

	// Close scope; opener becomes disabled.
	rt.Update(viewND(bg, disabledFocusableND()))

	// opener is disabled, focus falls back to bg.
	if rt.focused != rt.root.children[0] {
		t.Fatal("focus should fall back to bg when restore target is disabled")
	}
}

// TestHandleMouseActionDerivation verifies Ticket 3: runtime derives ActionPress,
// ActionMotion, and ActionRelease from lastMouseButtons state, not from event.Mouse.Action.
func TestHandleMouseActionDerivation(t *testing.T) {
	var actions []input.MouseAction
	rt := New()
	nd := &node.Node{
		Kind: node.ViewKind,
		OnMouse: func(mp input.MousePress) bool {
			actions = append(actions, mp.Action)
			return true
		},
	}
	rt.Update(nd)
	rt.RunLayout(80, 24)

	// First left-button event: button state changed from none → left → ActionPress.
	rt.handleMouse(event.Mouse{X: 0, Y: 0, Button: input.MouseLeft})
	// Same button mask, different position: ActionMotion.
	rt.handleMouse(event.Mouse{X: 5, Y: 0, Button: input.MouseLeft})
	// Button released (mask back to none): ActionRelease.
	rt.handleMouse(event.Mouse{X: 5, Y: 0, Button: input.MouseNone})

	if len(actions) != 3 {
		t.Fatalf("expected 3 mouse actions, got %d", len(actions))
	}
	if actions[0] != input.ActionPress {
		t.Errorf("action[0]: expected ActionPress, got %v", actions[0])
	}
	if actions[1] != input.ActionMotion {
		t.Errorf("action[1]: expected ActionMotion, got %v", actions[1])
	}
	if actions[2] != input.ActionRelease {
		t.Errorf("action[2]: expected ActionRelease, got %v", actions[2])
	}
}

func TestRunLayoutComponentRootFlexGrowParticipatesInParentLayout(t *testing.T) {
	compFn := func() *node.Node {
		return &node.Node{
			Kind:  node.ViewKind,
			Style: style.Style{FlexGrow: 1},
		}
	}

	root := &node.Node{
		Kind:  node.ViewKind,
		Style: style.Style{Direction: style.Row},
		Children: []*node.Node{
			{Kind: node.ComponentKind, CompFn: compFn, CompID: 123},
			{Kind: node.ViewKind, Style: style.Style{Width: style.Cells(10)}},
		},
	}

	rt := New()
	rt.Update(root)
	rt.RunLayout(80, 24)

	first := rt.root.children[0].layout.Rect
	second := rt.root.children[1].layout.Rect

	if first.W != 70 {
		t.Fatalf("component layout width = %d, want 70", first.W)
	}
	if second.X != 70 {
		t.Fatalf("fixed sibling X = %d, want 70", second.X)
	}
}
