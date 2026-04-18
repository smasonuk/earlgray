// Package node defines the internal node tree types used throughout the runtime.
package node

import (
	"github.com/smason/earlgray/internal/style"
)

// Kind identifies what type of node this is.
type Kind int

const (
	ViewKind      Kind = iota // a container with style and children
	TextKind                  // a leaf node with text content
	KeyedKind                 // wraps another node with an explicit key
	ComponentKind             // a function component
)

// TextAlign controls text alignment within its container.
type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

// TextOptions holds options for text nodes.
type TextOptions struct {
	Align TextAlign
}

// Node is the internal concrete node type.
type Node struct {
	Kind     Kind
	Key      string       // optional explicit key for reconciliation
	Style    style.Style  // style (ViewKind)
	Children []*Node      // child nodes (ViewKind)
	Text     string       // text content (TextKind)
	TextOpts TextOptions  // text options (TextKind)
	CompFn   func() *Node // component render function (ComponentKind)
	CompID   uintptr      // identity of component function (for reconciliation)
}
