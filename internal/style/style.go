// Package style defines layout and visual style primitives shared between
// the internal packages and the public API.
package style

import (
	"github.com/smason/earlgray/internal/color"
)

// Point represents a 2D coordinate.
type Point struct{ X, Y int }

// Size represents width and height.
type Size struct{ W, H int }

// Rect represents a rectangle with position and size.
type Rect struct{ X, Y, W, H int }

// Right returns the x coordinate one past the right edge.
func (r Rect) Right() int { return r.X + r.W }

// Bottom returns the y coordinate one past the bottom edge.
func (r Rect) Bottom() int { return r.Y + r.H }

// Contains reports whether the point is inside the rect.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}

// Inner returns the rect shrunk by the given insets.
func (r Rect) Inner(insets Insets) Rect {
	x := r.X + insets.Left
	y := r.Y + insets.Top
	w := r.W - insets.Left - insets.Right
	h := r.H - insets.Top - insets.Bottom
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, W: w, H: h}
}

// Intersect returns the intersection of two rects (may be empty).
func (r Rect) Intersect(other Rect) Rect {
	x1 := max2(r.X, other.X)
	y1 := max2(r.Y, other.Y)
	x2 := min2(r.Right(), other.Right())
	y2 := min2(r.Bottom(), other.Bottom())
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Insets represents top/right/bottom/left spacing.
type Insets struct{ Top, Right, Bottom, Left int }

// All returns uniform insets with the same value on all sides.
func All(n int) Insets {
	return Insets{Top: n, Right: n, Bottom: n, Left: n}
}

// DimKind distinguishes between auto and fixed dimensions.
type DimKind int

const (
	DimAuto  DimKind = iota
	DimCells         // fixed number of terminal cells
)

// Dimension is a layout dimension, either auto or a fixed cell count.
type Dimension struct {
	Kind  DimKind
	Value int
}

// Cells returns a fixed-size Dimension of n terminal cells.
func Cells(n int) Dimension {
	return Dimension{Kind: DimCells, Value: n}
}

// Auto returns an auto-sized Dimension.
func Auto() Dimension {
	return Dimension{Kind: DimAuto}
}

// Border describes which sides of a view have a border drawn.
type Border struct {
	Top, Bottom, Left, Right bool
}

// Border presets.
var (
	BorderNone   = Border{}
	BorderAll    = Border{Top: true, Bottom: true, Left: true, Right: true}
	BorderBottom = Border{Bottom: true}
	BorderRight  = Border{Right: true}
	BorderTop    = Border{Top: true}
	BorderLeft   = Border{Left: true}
)

// Insets returns the insets consumed by this border configuration.
func (b Border) Insets() Insets {
	var ins Insets
	if b.Top {
		ins.Top = 1
	}
	if b.Bottom {
		ins.Bottom = 1
	}
	if b.Left {
		ins.Left = 1
	}
	if b.Right {
		ins.Right = 1
	}
	return ins
}

// Direction is the flex direction.
type Direction int

const (
	Row    Direction = iota // children laid out horizontally
	Column                  // children laid out vertically
)

// Align controls how items are aligned on the cross axis.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// Justify controls how items are distributed on the main axis.
type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

// Style defines the visual and layout properties of a node.
type Style struct {
	Width, Height       Dimension
	MinWidth, MinHeight int
	MaxWidth, MaxHeight int
	FlexGrow            int
	// FlexShrink is reserved for future use and is not currently implemented.
	FlexShrink int
	Direction  Direction
	AlignItems Align
	Justify    Justify
	Padding    Insets
	Gap        int
	Border     Border
	Foreground color.Color
	Background color.Color
	Bold       bool
	Italic     bool
	Underline  bool
}

// Merge returns a new style that inherits unspecified colors from parent.
// Child values that are specified override parent values. Boolean attributes
// are not inherited (Go bools cannot distinguish false from unset).
func Merge(parent, child Style) Style {
	out := child

	if !out.Foreground.IsSpecified() {
		out.Foreground = parent.Foreground
	}
	if !out.Background.IsSpecified() {
		out.Background = parent.Background
	}

	return out
}
