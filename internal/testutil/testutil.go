// Package testutil provides helpers for testing buffer contents.
package testutil

import (
	"strings"

	"github.com/smasonuk/earlgray/internal/screen"
)

// BufferToGrid converts a Buffer to a slice of strings, one per row.
// Each string has exactly buf.W characters.
func BufferToGrid(buf *screen.Buffer) []string {
	rows := make([]string, buf.H)
	for y := 0; y < buf.H; y++ {
		var sb strings.Builder
		for x := 0; x < buf.W; x++ {
			r := buf.At(x, y).Rune
			if r == 0 {
				sb.WriteByte(' ')
			} else {
				sb.WriteRune(r)
			}
		}
		rows[y] = sb.String()
	}
	return rows
}

// GridString converts a Buffer to a single string with newlines between rows.
func GridString(buf *screen.Buffer) string {
	return strings.Join(BufferToGrid(buf), "\n")
}

// GridAt returns the rune at (x, y) in the buffer, or ' ' if empty.
func GridAt(buf *screen.Buffer, x, y int) rune {
	if x < 0 || x >= buf.W || y < 0 || y >= buf.H {
		return 0
	}
	r := buf.At(x, y).Rune
	if r == 0 {
		return ' '
	}
	return r
}
