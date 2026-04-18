// Package node defines the internal node tree types used throughout the runtime.
package node

import (
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/style"
)

// Kind identifies what type of node this is.
type Kind int

const (
	ViewKind      Kind = iota // a container with style and children
	TextKind                  // a leaf node with text content
	KeyedKind                 // wraps another node with an explicit key
	ComponentKind             // a function component
	OverlayKind               // stacks children on top of each other
)

// TextAlign controls text alignment within its container.
type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

// KeyPress holds data delivered to an OnKey handler.
type KeyPress = input.KeyPress

// KeyHandler processes a key press. Returns true if the event was consumed.
type KeyHandler func(KeyPress) bool

// TextOptions holds options for text nodes.
type TextOptions struct {
	Align TextAlign
	Style style.Style
}

// Node is the internal concrete node type.
type Node struct {
	Kind      Kind
	Key       string       // optional explicit key for reconciliation
	Style     style.Style  // style (ViewKind)
	Children  []*Node      // child nodes (ViewKind, KeyedKind)
	Text      string       // text content (TextKind)
	TextOpts  TextOptions  // text options (TextKind)
	CompFn    func() *Node // component render function (ComponentKind)
	CompID    uintptr      // identity of component function (for reconciliation)
	OnKey     KeyHandler   // optional key handler (ViewKind)
	Focusable bool         // whether this node can receive focus
	AutoFocus bool         // request focus on initial mount if no other node is focused
	Disabled  bool         // skip in focus traversal and key delivery

	// FocusScope traps focus traversal within this view's subtree.
	FocusScope bool

	// Cursor request: if CursorVisible is true, the runtime will show the
	// terminal cursor at (CursorX, CursorY) relative to the node's content rect.
	CursorVisible bool
	CursorX       int
	CursorY       int
}
