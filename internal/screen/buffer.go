// Package screen implements a cell buffer for terminal rendering.
package screen

import (
	"github.com/mattn/go-runewidth"
	"github.com/smason/earlgray/internal/color"
)

// CellStyle describes text attributes for a single cell.
type CellStyle struct {
	Fg, Bg    color.Color
	Bold      bool
	Italic    bool
	Underline bool
}

// Cell is a single terminal cell with a rune and its style.
type Cell struct {
	Rune  rune
	Style CellStyle
}

// Buffer is a 2D grid of cells, row-major order.
type Buffer struct {
	W, H  int
	Cells []Cell
}

// NewBuffer creates a Buffer of the given size, filled with spaces.
func NewBuffer(w, h int) *Buffer {
	cells := make([]Cell, w*h)
	for i := range cells {
		cells[i].Rune = ' '
	}
	return &Buffer{W: w, H: h, Cells: cells}
}

// index returns the slice index for (x, y).
func (b *Buffer) index(x, y int) int {
	return y*b.W + x
}

// inBounds reports whether (x, y) is within the buffer.
func (b *Buffer) inBounds(x, y int) bool {
	return x >= 0 && x < b.W && y >= 0 && y < b.H
}

// At returns the cell at (x, y). Panics if out of bounds.
func (b *Buffer) At(x, y int) Cell {
	return b.Cells[b.index(x, y)]
}

// SetCell sets the cell at (x, y) to ch with the given style.
// Silently ignores out-of-bounds writes.
func (b *Buffer) SetCell(x, y int, ch rune, style CellStyle) {
	if !b.inBounds(x, y) {
		return
	}
	b.Cells[b.index(x, y)] = Cell{Rune: ch, Style: style}
}

// Clear fills the entire buffer with spaces using the default style.
func (b *Buffer) Clear() {
	for i := range b.Cells {
		b.Cells[i] = Cell{Rune: ' '}
	}
}

// FillRect fills a rectangular area with the given rune and style.
func (b *Buffer) FillRect(x, y, w, h int, ch rune, style CellStyle) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			b.SetCell(col, row, ch, style)
		}
	}
}

// DrawHLine draws a horizontal line from (x, y) of length w.
func (b *Buffer) DrawHLine(x, y, w int, ch rune, style CellStyle) {
	for col := x; col < x+w; col++ {
		b.SetCell(col, y, ch, style)
	}
}

// DrawVLine draws a vertical line from (x, y) of height h.
func (b *Buffer) DrawVLine(x, y, h int, ch rune, style CellStyle) {
	for row := y; row < y+h; row++ {
		b.SetCell(x, row, ch, style)
	}
}

// DrawTextClipped draws text at (x, y) clipped to the rectangle (cx, cy, cw, ch).
// Returns the x position after the last character drawn.
func (b *Buffer) DrawTextClipped(x, y int, text string, style CellStyle, clipX, clipY, clipW, clipH int) int {
	col := x
	clipRight := clipX + clipW
	clipBottom := clipY + clipH
	if y < clipY || y >= clipBottom {
		return col
	}
	for _, r := range text {
		w := runewidth.RuneWidth(r)
		if w == 0 {
			// combining character, skip for simplicity
			continue
		}
		// Don't draw partial wide characters at clip boundary
		if col+w > clipRight {
			break
		}
		if col >= clipX {
			b.SetCell(col, y, r, style)
			// For wide characters, fill the next cell with a space
			if w > 1 {
				b.SetCell(col+1, y, ' ', style)
			}
		}
		col += w
	}
	return col
}
