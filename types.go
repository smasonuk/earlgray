// Package tui provides a React-like terminal UI library.
package tui

import (
	icolor "github.com/smason/earlgray/internal/color"
	istyle "github.com/smason/earlgray/internal/style"
)

// Point represents a 2D coordinate.
type Point = istyle.Point

// Size represents width and height.
type Size = istyle.Size

// Rect represents a rectangle with position and size.
type Rect = istyle.Rect

// Insets represents top/right/bottom/left spacing.
type Insets = istyle.Insets

// All returns uniform insets with the same value on all sides.
func All(n int) Insets { return istyle.All(n) }

// DimKind distinguishes between auto and fixed dimensions.
type DimKind = istyle.DimKind

const (
	DimAuto  = istyle.DimAuto
	DimCells = istyle.DimCells
)

// Dimension is a layout dimension.
type Dimension = istyle.Dimension

// Cells returns a fixed-size Dimension of n terminal cells.
func Cells(n int) Dimension { return istyle.Cells(n) }

// Auto returns an auto-sized Dimension.
func Auto() Dimension { return istyle.Auto() }

// Direction is the flex direction.
type Direction = istyle.Direction

const (
	Row    = istyle.Row
	Column = istyle.Column
)

// Align controls cross-axis alignment.
type Align = istyle.Align

const (
	AlignStart   = istyle.AlignStart
	AlignCenter  = istyle.AlignCenter
	AlignEnd     = istyle.AlignEnd
	AlignStretch = istyle.AlignStretch
)

// Justify controls main-axis distribution.
type Justify = istyle.Justify

const (
	JustifyStart        = istyle.JustifyStart
	JustifyCenter       = istyle.JustifyCenter
	JustifyEnd          = istyle.JustifyEnd
	JustifySpaceBetween = istyle.JustifySpaceBetween
)

// Border describes which sides have a border.
type Border = istyle.Border

// Border presets.
var (
	BorderNone   = istyle.BorderNone
	BorderAll    = istyle.BorderAll
	BorderBottom = istyle.BorderBottom
	BorderRight  = istyle.BorderRight
	BorderTop    = istyle.BorderTop
	BorderLeft   = istyle.BorderLeft
)

// Color is a terminal color.
// Use DefaultColor(), ANSIColor(), or RGBColor() to create values.
type Color = icolor.Color

// DefaultColor returns the terminal default color.
func DefaultColor() Color { return icolor.DefaultColor() }

// ANSIColor returns one of the 16 ANSI palette colors (0-15).
func ANSIColor(n int) Color { return icolor.ANSIColor(n) }

// RGBColor returns a 24-bit RGB color.
func RGBColor(r, g, b uint8) Color { return icolor.RGBColor(r, g, b) }

// Style defines the visual and layout properties of a node.
type Style = istyle.Style
