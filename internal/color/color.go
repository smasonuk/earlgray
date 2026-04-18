// Package color provides internal color representation shared between packages.
package color

import "github.com/gdamore/tcell/v2"

// Kind identifies the type of color.
type Kind int

const (
	Unspecified Kind = iota // no color specified; inherit from parent
	Default                 // explicitly use terminal default color
	ANSI
	RGB
)

// Color is a terminal color.
type Color struct {
	Kind    Kind
	ANSIVal int
	R, G, B uint8
}

// DefaultColor returns the terminal default color.
func DefaultColor() Color {
	return Color{Kind: Default}
}

// ANSIColor returns a terminal palette color (0-255).
func ANSIColor(n int) Color {
	return Color{Kind: ANSI, ANSIVal: n}
}

// RGBColor returns a 24-bit RGB color.
func RGBColor(r, g, b uint8) Color {
	return Color{Kind: RGB, R: r, G: g, B: b}
}

// IsSpecified reports whether a color is explicitly specified.
// Unspecified colors should inherit from parent; specified colors should override.
func (c Color) IsSpecified() bool {
	return c.Kind != Unspecified
}

// ToTcell converts the color to a tcell.Color.
func (c Color) ToTcell() tcell.Color {
	switch c.Kind {
	case Default, Unspecified:
		return tcell.ColorDefault
	case ANSI:
		return tcell.PaletteColor(c.ANSIVal)
	case RGB:
		return tcell.NewRGBColor(int32(c.R), int32(c.G), int32(c.B))
	}
	return tcell.ColorDefault
}
