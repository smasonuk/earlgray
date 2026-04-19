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
	RichTextKind              // a leaf node with multiple styled text spans
	KeyedKind                 // wraps another node with an explicit key
	ComponentKind             // a function component
	OverlayKind               // stacks children on top of each other
	TextPanelKind             // a scrollable text panel
	TextAreaKind              // editable multi-line text area
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

// KeyCaptureHandler processes a key press during the capture phase.
type KeyCaptureHandler func(KeyPress) bool

// MouseHandler processes a mouse event. Returns true if the event was consumed.
type MouseHandler func(input.MousePress) bool

// TextOptions holds options for text nodes.
type TextOptions struct {
	Align TextAlign
	Style style.Style
}

// TextSpan is a styled segment in a RichText node.
type TextSpan struct {
	Text  string
	Style style.Style
}

// TextPanelOptions holds options for scrollable text panels.
type TextPanelOptions struct {
	WordWrap         bool
	ShowScrollbar    bool
	AutoScrollBottom bool
	ResetScrollKey   string
	InitialScrollX   int
	InitialScrollY   int
}

// TextAreaOptions holds options for editable multi-line text areas.
type TextAreaOptions struct {
	Placeholder string

	// WordWrap wraps long logical lines to the textarea content width.
	// When false, long lines are horizontally clipped and Left/Right scrolling is used.
	WordWrap bool

	// ShowScrollbar draws a vertical scrollbar when content exceeds the viewport.
	ShowScrollbar bool

	// OnChange receives edited text. If nil, edit keys are ignored.
	OnChange func(string)

	// OnSubmit is called by Ctrl+Enter when SubmitOnCtrlEnter is true.
	// Plain Enter inserts a newline.
	OnSubmit func(string)

	SubmitOnCtrlEnter bool
}

// Node is the internal concrete node type.
type Node struct {
	Kind          Kind
	Key           string           // optional explicit key for reconciliation
	Style         style.Style      // style (ViewKind)
	Children      []*Node          // child nodes (ViewKind, KeyedKind)
	Text          string           // text content (TextKind)
	Spans         []TextSpan       // rich text spans (RichTextKind)
	TextOpts      TextOptions      // text options (TextKind)
	TextPanelOpts TextPanelOptions // text panel options (TextPanelKind)
	TextAreaOpts  TextAreaOptions  // text area options (TextAreaKind)
	CompFn        func() *Node     // component render function (ComponentKind)
	CompID        uintptr          // identity of component function (for reconciliation)
	OnKey         KeyHandler       // optional key handler (ViewKind)
	OnKeyCapture  KeyCaptureHandler
	OnMouse       MouseHandler // optional mouse handler (ViewKind)
	Focusable     bool         // whether this node can receive focus
	AutoFocus     bool         // request focus on initial mount if no other node is focused
	Disabled      bool         // skip in focus traversal and key delivery

	// FocusScope traps focus traversal within this view's subtree.
	FocusScope bool

	// Cursor request: if CursorVisible is true, the runtime will show the
	// terminal cursor at (CursorX, CursorY) relative to the node's content rect.
	CursorVisible bool
	CursorX       int
	CursorY       int
}
