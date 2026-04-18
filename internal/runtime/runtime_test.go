package runtime

import (
	"testing"

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
