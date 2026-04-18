// Package event defines internal event types for the TUI runtime.
package event

import "github.com/gdamore/tcell/v2"

// Kind identifies the type of event.
type Kind int

const (
	KeyKind    Kind = iota
	ResizeKind      // terminal was resized
	FocusKind       // terminal gained focus
	BlurKind        // terminal lost focus
	QuitKind        // quit signal
)

// Key holds key event data.
type Key struct {
	Key  tcell.Key     // key code
	Rune rune          // rune for printable keys
	Mod  tcell.ModMask // modifier keys
}

// Event is a unified internal event.
type Event struct {
	Kind   Kind
	Key    Key // valid if Kind == KeyKind
	Width  int // valid if Kind == ResizeKind
	Height int
}
