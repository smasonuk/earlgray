// Package host defines the Host interface for terminal backend abstraction.
package host

import (
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/screen"
)

// Host is the interface implemented by terminal backends.
type Host interface {
	// Init initializes the terminal.
	Init() error
	// Close restores the terminal to its original state.
	Close() error
	// Size returns the current terminal dimensions.
	Size() (w, h int)
	// PollEvent blocks until an event is available and returns it.
	PollEvent() event.Event
	// Show flushes pending changes to the terminal.
	Show()
	// Sync forces a full redraw.
	Sync()
	// SetCell draws a single cell at (x, y).
	SetCell(x, y int, ch rune, style screen.CellStyle)
	// ShowCursor positions the cursor at (x, y).
	ShowCursor(x, y int)
	// HideCursor hides the terminal cursor.
	HideCursor()
}
