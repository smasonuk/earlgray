// Package render paints the instance tree into a cell buffer.
// This is a thin wrapper; the actual painting logic lives in runtime.renderInstance.
// This package exports helper utilities for external use if needed.
package render

import (
	"github.com/smason/earlgray/internal/screen"
)

// FlushDiff compares two buffers and sends all changes to the Differ.
func FlushDiff(prev, next *screen.Buffer, d screen.Differ) {
	screen.Diff(prev, next, d)
}
