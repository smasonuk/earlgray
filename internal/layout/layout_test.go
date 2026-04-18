package layout

import (
	"testing"

	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/style"
)

// helper: build a view node.
func viewNode(s style.Style, children ...*node.Node) *node.Node {
	return &node.Node{Kind: node.ViewKind, Style: s, Children: children}
}

// helper: build a text node.
func textNode() *node.Node {
	return &node.Node{Kind: node.TextKind}
}

// helper: root constraints filling 80x24.
func rootC() Constraints { return Constraints{MinW: 0, MaxW: 80, MinH: 0, MaxH: 24} }

// helper: exact constraints.
func exactC(w, h int) Constraints { return Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h} }

func TestSingleViewFillsRoot(t *testing.T) {
	n := viewNode(style.Style{})
	tree := Layout(n, rootC())
	want := style.Rect{X: 0, Y: 0, W: 80, H: 24}
	if tree.Result.Rect != want {
		t.Errorf("rect: got %+v, want %+v", tree.Result.Rect, want)
	}
}

func TestFixedWidthHeight(t *testing.T) {
	n := viewNode(style.Style{
		Width:  style.Cells(40),
		Height: style.Cells(10),
	})
	tree := Layout(n, rootC())
	want := style.Rect{X: 0, Y: 0, W: 40, H: 10}
	if tree.Result.Rect != want {
		t.Errorf("rect: got %+v, want %+v", tree.Result.Rect, want)
	}
}

func TestRowTwoFixedChildren(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row},
		viewNode(style.Style{Width: style.Cells(20)}),
		viewNode(style.Style{Width: style.Cells(30)}),
	)
	tree := Layout(parent, exactC(80, 24))
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree.Children))
	}
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	if c0 != (style.Rect{X: 0, Y: 0, W: 20, H: 24}) {
		t.Errorf("child 0: got %+v", c0)
	}
	if c1 != (style.Rect{X: 20, Y: 0, W: 30, H: 24}) {
		t.Errorf("child 1: got %+v", c1)
	}
}

func TestColumnTwoFixedChildren(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Column},
		viewNode(style.Style{Height: style.Cells(5)}),
		viewNode(style.Style{Height: style.Cells(10)}),
	)
	tree := Layout(parent, exactC(80, 24))
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree.Children))
	}
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	if c0 != (style.Rect{X: 0, Y: 0, W: 80, H: 5}) {
		t.Errorf("child 0: got %+v", c0)
	}
	if c1 != (style.Rect{X: 0, Y: 5, W: 80, H: 10}) {
		t.Errorf("child 1: got %+v", c1)
	}
}

func TestFlexGrowSingleChild(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row},
		viewNode(style.Style{FlexGrow: 1}),
	)
	tree := Layout(parent, exactC(80, 24))
	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree.Children))
	}
	c := tree.Children[0].Result.Rect
	if c != (style.Rect{X: 0, Y: 0, W: 80, H: 24}) {
		t.Errorf("flex child: got %+v", c)
	}
}

func TestFlexGrowTwoChildren(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row},
		viewNode(style.Style{FlexGrow: 1}),
		viewNode(style.Style{FlexGrow: 1}),
	)
	tree := Layout(parent, exactC(80, 24))
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree.Children))
	}
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	if c0.X != 0 || c0.W != 40 {
		t.Errorf("child 0: got %+v", c0)
	}
	if c1.X != 40 || c1.W != 40 {
		t.Errorf("child 1: got %+v", c1)
	}
}

func TestFlexGrowUnequalRatio(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row},
		viewNode(style.Style{FlexGrow: 1}),
		viewNode(style.Style{FlexGrow: 3}),
	)
	tree := Layout(parent, exactC(80, 24))
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	// 80 total: 1/4 = 20, 3/4 = 60
	if c0.W != 20 {
		t.Errorf("child 0 width: got %d, want 20", c0.W)
	}
	if c1.W != 60 {
		t.Errorf("child 1 width: got %d, want 60", c1.W)
	}
	if c1.X != 20 {
		t.Errorf("child 1 x: got %d, want 20", c1.X)
	}
}

func TestFlexGrowWithFixed(t *testing.T) {
	// Row: 20px fixed + flex=1 fills rest (80-20=60).
	parent := viewNode(
		style.Style{Direction: style.Row},
		viewNode(style.Style{Width: style.Cells(20)}),
		viewNode(style.Style{FlexGrow: 1}),
	)
	tree := Layout(parent, exactC(80, 24))
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	if c0.W != 20 {
		t.Errorf("fixed child width: got %d, want 20", c0.W)
	}
	if c1.X != 20 || c1.W != 60 {
		t.Errorf("flex child: got %+v", c1)
	}
}

func TestGapBetweenChildren(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, Gap: 2},
		viewNode(style.Style{Width: style.Cells(10)}),
		viewNode(style.Style{Width: style.Cells(10)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c1 := tree.Children[1].Result.Rect
	// Second child should start at 10 (first) + 2 (gap) = 12.
	if c1.X != 12 {
		t.Errorf("child 1 x: got %d, want 12", c1.X)
	}
}

func TestPaddingReducesContent(t *testing.T) {
	n := viewNode(style.Style{
		Width:   style.Cells(20),
		Height:  style.Cells(10),
		Padding: style.All(2),
	})
	tree := Layout(n, rootC())
	content := tree.Result.Content
	want := style.Rect{X: 2, Y: 2, W: 16, H: 6}
	if content != want {
		t.Errorf("content: got %+v, want %+v", content, want)
	}
}

func TestBorderReducesContent(t *testing.T) {
	n := viewNode(style.Style{
		Width:  style.Cells(10),
		Height: style.Cells(5),
		Border: style.BorderAll,
	})
	tree := Layout(n, rootC())
	content := tree.Result.Content
	// BorderAll reduces each side by 1.
	want := style.Rect{X: 1, Y: 1, W: 8, H: 3}
	if content != want {
		t.Errorf("content: got %+v, want %+v", content, want)
	}
}

func TestCrossAxisAlignCenter(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, AlignItems: style.AlignCenter},
		viewNode(style.Style{Width: style.Cells(10), Height: style.Cells(4)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c := tree.Children[0].Result.Rect
	// Cross axis is vertical: (24-4)/2 = 10.
	if c.Y != 10 {
		t.Errorf("child Y: got %d, want 10", c.Y)
	}
	if c.H != 4 {
		t.Errorf("child H: got %d, want 4", c.H)
	}
}

func TestCrossAxisAlignEnd(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, AlignItems: style.AlignEnd},
		viewNode(style.Style{Width: style.Cells(10), Height: style.Cells(6)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c := tree.Children[0].Result.Rect
	// Cross axis is vertical: 24-6 = 18.
	if c.Y != 18 {
		t.Errorf("child Y: got %d, want 18", c.Y)
	}
}

func TestJustifyCenter(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, Justify: style.JustifyCenter},
		viewNode(style.Style{Width: style.Cells(20)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c := tree.Children[0].Result.Rect
	// Free space = 80-20 = 60, center offset = 30.
	if c.X != 30 {
		t.Errorf("child X: got %d, want 30", c.X)
	}
}

func TestJustifyEnd(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, Justify: style.JustifyEnd},
		viewNode(style.Style{Width: style.Cells(20)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c := tree.Children[0].Result.Rect
	// Free space = 80-20 = 60, end offset = 60.
	if c.X != 60 {
		t.Errorf("child X: got %d, want 60", c.X)
	}
}

func TestJustifySpaceBetween(t *testing.T) {
	parent := viewNode(
		style.Style{Direction: style.Row, Justify: style.JustifySpaceBetween},
		viewNode(style.Style{Width: style.Cells(10)}),
		viewNode(style.Style{Width: style.Cells(10)}),
	)
	tree := Layout(parent, exactC(80, 24))
	c0 := tree.Children[0].Result.Rect
	c1 := tree.Children[1].Result.Rect
	// Free space = 80-20 = 60, spacing between = 60.
	if c0.X != 0 {
		t.Errorf("child 0 X: got %d, want 0", c0.X)
	}
	if c1.X != 70 {
		t.Errorf("child 1 X: got %d, want 70", c1.X)
	}
}

func TestNestedLayout(t *testing.T) {
	// Column: header(h=1) + row(flex=1 containing sidebar(w=24) + main(flex=1))
	root := viewNode(
		style.Style{Direction: style.Column},
		viewNode(style.Style{Height: style.Cells(1)}),
		viewNode(
			style.Style{Direction: style.Row, FlexGrow: 1},
			viewNode(style.Style{Width: style.Cells(24)}),
			viewNode(style.Style{FlexGrow: 1}),
		),
	)
	tree := Layout(root, exactC(80, 24))
	if len(tree.Children) != 2 {
		t.Fatalf("root children: %d", len(tree.Children))
	}
	header := tree.Children[0].Result.Rect
	rowNode := tree.Children[1].Result.Rect
	if header != (style.Rect{X: 0, Y: 0, W: 80, H: 1}) {
		t.Errorf("header: %+v", header)
	}
	if rowNode != (style.Rect{X: 0, Y: 1, W: 80, H: 23}) {
		t.Errorf("row: %+v", rowNode)
	}
	rowChildren := tree.Children[1].Children
	if len(rowChildren) != 2 {
		t.Fatalf("row children: %d", len(rowChildren))
	}
	sidebar := rowChildren[0].Result.Rect
	main := rowChildren[1].Result.Rect
	if sidebar != (style.Rect{X: 0, Y: 1, W: 24, H: 23}) {
		t.Errorf("sidebar: %+v", sidebar)
	}
	if main != (style.Rect{X: 24, Y: 1, W: 56, H: 23}) {
		t.Errorf("main: %+v", main)
	}
}
