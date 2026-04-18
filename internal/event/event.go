// Package event defines internal event types for the TUI runtime.
package event

import (
	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/input"
)

// Kind identifies the type of event.
type Kind int

const (
	KeyKind    Kind = iota
	ResizeKind      // terminal was resized
	FocusKind       // terminal gained focus
	BlurKind        // terminal lost focus
	QuitKind        // quit signal
	MouseKind       // mouse button or wheel event
)

// Key holds key event data.
type Key struct {
	Key  tcell.Key     // key code
	Rune rune          // rune for printable keys
	Mod  tcell.ModMask // modifier keys
}

// IsTab reports whether this key event is a Tab key press (forward traversal).
func (k Key) IsTab() bool {
	return k.Key == tcell.KeyTab && k.Mod&tcell.ModShift == 0
}

// IsShiftTab reports whether this key event is a Shift+Tab (reverse traversal).
// Terminals may report this as KeyBacktab or as KeyTab with ModShift.
func (k Key) IsShiftTab() bool {
	return k.Key == tcell.KeyBacktab || (k.Key == tcell.KeyTab && k.Mod&tcell.ModShift != 0)
}

// IsCtrlC reports whether this key event is Ctrl-C.
func (k Key) IsCtrlC() bool {
	if k.Key == tcell.KeyCtrlC {
		return true
	}
	if k.Key == tcell.KeyRune && (k.Rune == 'c' || k.Rune == 'C') && k.Mod&tcell.ModCtrl != 0 {
		return true
	}
	return false
}

// NormalizeKey converts a tcell key to a shared Key enum value.
// Returns KeyUnknown for unrecognized keys.
func NormalizeKey(tcellKey tcell.Key, rune rune) input.Key {
	if rune != 0 && tcellKey != tcell.KeyRune {
		return input.KeyRune
	}

	switch tcellKey {
	case tcell.KeyRune:
		if rune != 0 {
			return input.KeyRune
		}
		return input.KeyUnknown
	case tcell.KeyEnter:
		return input.KeyEnter
	case tcell.KeyEsc:
		return input.KeyEsc
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return input.KeyBackspace
	case tcell.KeyTab:
		return input.KeyTab
	case tcell.KeyUp:
		return input.KeyUp
	case tcell.KeyDown:
		return input.KeyDown
	case tcell.KeyLeft:
		return input.KeyLeft
	case tcell.KeyRight:
		return input.KeyRight
	case tcell.KeyHome:
		return input.KeyHome
	case tcell.KeyEnd:
		return input.KeyEnd
	case tcell.KeyPgUp:
		return input.KeyPgUp
	case tcell.KeyPgDn:
		return input.KeyPgDown
	case tcell.KeyDelete:
		return input.KeyDelete
	case tcell.KeyInsert:
		return input.KeyInsert
	default:
		return input.KeyUnknown
	}
}

// Mouse holds mouse event data.
type Mouse struct {
	X, Y   int
	Button input.MouseButton
	Action input.MouseAction
	Mod    tcell.ModMask
}

// Event is a unified internal event.
type Event struct {
	Kind   Kind
	Key    Key   // valid if Kind == KeyKind
	Mouse  Mouse // valid if Kind == MouseKind
	Width  int   // valid if Kind == ResizeKind
	Height int
}
