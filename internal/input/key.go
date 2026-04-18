// Package input defines key and modifier types used throughout the TUI.
package input

// Key identifies a special key or KeyRune for printable characters.
type Key int

const (
	KeyUnknown Key = iota
	KeyRune              // printable character in KeyPress.Rune
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyTab
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyDelete
	KeyInsert
)

// Mod is a modifier key bitmask.
type Mod int

const (
	ModNone  Mod = 0
	ModCtrl  Mod = 1 << 0
	ModAlt   Mod = 1 << 1
	ModShift Mod = 1 << 2
)

// KeyPress holds data delivered to a key handler.
type KeyPress struct {
	Key  Key
	Rune rune
	Mod  Mod
}
