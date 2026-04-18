// Package input defines key and modifier types used throughout the TUI.
package input

// Key identifies a special key or KeyRune for printable characters.
type Key int

const (
	KeyUnknown Key = iota
	KeyRune        // printable character in KeyPress.Rune
	KeyCtrlC
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

// MouseButton identifies which mouse button or wheel direction was activated.
type MouseButton int

const (
	MouseNone      MouseButton = 0
	MouseLeft      MouseButton = 1 << 0
	MouseMiddle    MouseButton = 1 << 1
	MouseRight     MouseButton = 1 << 2
	MouseWheelUp   MouseButton = 1 << 3
	MouseWheelDown MouseButton = 1 << 4
)

// MouseAction identifies the type of mouse activity.
type MouseAction int

const (
	ActionPress MouseAction = iota
	ActionRelease
	ActionMotion
)

// MousePress holds data delivered to an OnMouse handler.
type MousePress struct {
	X, Y           int
	LocalX, LocalY int
	Button         MouseButton
	Action         MouseAction
	Mod            Mod
}
