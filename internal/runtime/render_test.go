package runtime

import (
	"testing"

	"github.com/smason/earlgray/internal/color"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/screen"
	"github.com/smason/earlgray/internal/style"
)

// renderHelper sets up a runtime and renders to a buffer.
func renderHelper(nd *node.Node, w, h int) *screen.Buffer {
	rt := New()
	rt.Update(nd)
	rt.RunLayout(w, h)
	buf := screen.NewBuffer(w, h)
	rt.Render(buf)
	return buf
}

func TestRenderViewBorder(t *testing.T) {
	// Create a small bordered view with no children.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(5),
			Height: style.Cells(3),
			Border: style.BorderAll,
		},
	}
	buf := renderHelper(nd, 10, 10)

	// Check corners (should be unicode box drawing chars, but we'll just check not space)
	if buf.At(0, 0).Rune == ' ' {
		t.Error("top-left corner should not be space")
	}
	if buf.At(4, 0).Rune == ' ' {
		t.Error("top-right corner should not be space")
	}
	if buf.At(0, 2).Rune == ' ' {
		t.Error("bottom-left corner should not be space")
	}
	if buf.At(4, 2).Rune == ' ' {
		t.Error("bottom-right corner should not be space")
	}

	// Check top border
	if buf.At(1, 0).Rune == ' ' || buf.At(2, 0).Rune == ' ' || buf.At(3, 0).Rune == ' ' {
		t.Error("top border should not contain spaces")
	}

	// Check left border
	if buf.At(0, 1).Rune == ' ' {
		t.Error("left border should not be space")
	}
}

func TestRenderViewPadding(t *testing.T) {
	// Create a view with padding containing text.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:   style.Cells(7),
			Height:  style.Cells(3),
			Padding: style.All(1),
		},
		Children: []*node.Node{
			{Kind: node.TextKind, Text: "hi"},
		},
	}
	buf := renderHelper(nd, 10, 10)

	// Text should start at (1, 1) due to padding of 1
	if buf.At(1, 1).Rune != 'h' {
		t.Errorf("text should start at (1,1), got %q at (1,1)", buf.At(1, 1).Rune)
	}
	if buf.At(2, 1).Rune != 'i' {
		t.Errorf("expected 'i' at (2,1), got %q", buf.At(2, 1).Rune)
	}
}

func TestRenderCenteredText(t *testing.T) {
	// Create a text node with width 10 containing centered text "hi".
	// Expected: "h" at x=4, "i" at x=5 (centered in 10-cell width)
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(10),
			Height: style.Cells(1),
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "hi",
				Style: style.Style{
					Width: style.Cells(10),
				},
				TextOpts: node.TextOptions{
					Align: node.TextAlignCenter,
				},
			},
		},
	}
	buf := renderHelper(nd, 10, 1)

	// "hi" (2 cells) centered in 10 cells: (10-2)/2 = 4
	if buf.At(4, 0).Rune != 'h' {
		t.Errorf("centered 'h' should be at x=4, got %q at (4,0)", buf.At(4, 0).Rune)
	}
	if buf.At(5, 0).Rune != 'i' {
		t.Errorf("centered 'i' should be at x=5, got %q at (5,0)", buf.At(5, 0).Rune)
	}
}

func TestRenderRightAlignedText(t *testing.T) {
	// Create a text node with width 10 containing right-aligned text "x".
	// Expected: "x" at x=9 (right edge of 10-cell width)
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(10),
			Height: style.Cells(1),
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "x",
				Style: style.Style{
					Width: style.Cells(10),
				},
				TextOpts: node.TextOptions{
					Align: node.TextAlignRight,
				},
			},
		},
	}
	buf := renderHelper(nd, 10, 1)

	// "x" (1 cell) right-aligned in 10 cells: position 9
	if buf.At(9, 0).Rune != 'x' {
		t.Errorf("right-aligned 'x' should be at x=9, got %q at (9,0)", buf.At(9, 0).Rune)
	}
}

func TestRenderMultilineText(t *testing.T) {
	// Create a view with multiline text.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(10),
			Height: style.Cells(3),
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "a\nb\nc",
			},
		},
	}
	buf := renderHelper(nd, 10, 3)

	if buf.At(0, 0).Rune != 'a' {
		t.Errorf("line 0: expected 'a', got %q", buf.At(0, 0).Rune)
	}
	if buf.At(0, 1).Rune != 'b' {
		t.Errorf("line 1: expected 'b', got %q", buf.At(0, 1).Rune)
	}
	if buf.At(0, 2).Rune != 'c' {
		t.Errorf("line 2: expected 'c', got %q", buf.At(0, 2).Rune)
	}
}

func TestRenderTextStyle(t *testing.T) {
	// Create a text node with style.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(5),
			Height: style.Cells(1),
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "x",
				TextOpts: node.TextOptions{
					Style: style.Style{
						Foreground: color.ANSIColor(2),
						Bold:       true,
					},
				},
			},
		},
	}
	buf := renderHelper(nd, 5, 1)

	c := buf.At(0, 0)
	if c.Rune != 'x' {
		t.Errorf("expected 'x', got %q", c.Rune)
	}
	if c.Style.Fg.Kind != color.ANSI {
		t.Error("expected ANSI foreground color")
	}
	if !c.Style.Bold {
		t.Error("expected bold style")
	}
}

func TestRenderStyleInheritance(t *testing.T) {
	// Create a parent view with foreground color and a child text node without color.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:      style.Cells(10),
			Height:     style.Cells(1),
			Foreground: color.ANSIColor(3), // yellow parent
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "x",
				TextOpts: node.TextOptions{
					// No foreground specified, should inherit
				},
			},
		},
	}
	buf := renderHelper(nd, 10, 1)

	c := buf.At(0, 0)
	if c.Rune != 'x' {
		t.Errorf("expected 'x', got %q", c.Rune)
	}
	// Text should inherit parent's foreground
	if c.Style.Fg.Kind != color.ANSI || c.Style.Fg.ANSIVal != 3 {
		t.Errorf("expected inherited color ANSI(3), got %v", c.Style.Fg)
	}
}

func TestRenderStyleInheritanceOverride(t *testing.T) {
	// Create a parent view with foreground color and a child text with explicit color.
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:      style.Cells(10),
			Height:     style.Cells(1),
			Foreground: color.ANSIColor(3), // yellow parent
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "x",
				TextOpts: node.TextOptions{
					Style: style.Style{
						Foreground: color.ANSIColor(2), // green child
					},
				},
			},
		},
	}
	buf := renderHelper(nd, 10, 1)

	c := buf.At(0, 0)
	// Text should use its own color, not parent's
	if c.Style.Fg.Kind != color.ANSI || c.Style.Fg.ANSIVal != 2 {
		t.Errorf("expected child color ANSI(2), got %v", c.Style.Fg)
	}
}

func TestRenderWideCharacterClipping(t *testing.T) {
	// Create a text node with a wide character that might be clipped.
	// "界" is width 2, "a" is width 1
	nd := &node.Node{
		Kind: node.ViewKind,
		Style: style.Style{
			Width:  style.Cells(2),
			Height: style.Cells(1),
		},
		Children: []*node.Node{
			{
				Kind: node.TextKind,
				Text: "a界",
			},
		},
	}
	buf := renderHelper(nd, 2, 1)

	// "a" should be at (0,0)
	if buf.At(0, 0).Rune != 'a' {
		t.Errorf("expected 'a' at (0,0), got %q", buf.At(0, 0).Rune)
	}
	// Position 1 should be space, not partial "界"
	if buf.At(1, 0).Rune == '界' {
		t.Error("wide character should not be partially drawn at clip boundary")
	}
}
