package tui

import (
	"reflect"

	"github.com/mattn/go-runewidth"
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/host"
	"github.com/smason/earlgray/internal/input"
	inode "github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/render"
	"github.com/smason/earlgray/internal/runtime"
	"github.com/smason/earlgray/internal/screen"
)

// Node is the opaque tree element returned by View, Text, Keyed, etc.
type Node = *inode.Node

// Key identifies a special key or KeyRune for printable characters.
type Key = input.Key

const (
	KeyUnknown   = input.KeyUnknown
	KeyRune      = input.KeyRune
	KeyEnter     = input.KeyEnter
	KeyEsc       = input.KeyEsc
	KeyBackspace = input.KeyBackspace
	KeyTab       = input.KeyTab
	KeyUp        = input.KeyUp
	KeyDown      = input.KeyDown
	KeyLeft      = input.KeyLeft
	KeyRight     = input.KeyRight
	KeyHome      = input.KeyHome
	KeyEnd       = input.KeyEnd
	KeyPgUp      = input.KeyPgUp
	KeyPgDown    = input.KeyPgDown
	KeyDelete    = input.KeyDelete
	KeyInsert    = input.KeyInsert
)

// Mod is a modifier key bitmask.
type Mod = input.Mod

const (
	ModNone  = input.ModNone
	ModCtrl  = input.ModCtrl
	ModAlt   = input.ModAlt
	ModShift = input.ModShift
)

// KeyEvent holds data for a key handler.
type KeyEvent = input.KeyPress

// TextOption configures a Text node.
type TextOption func(*inode.TextOptions)

// WithAlign sets text alignment.
func WithAlign(a Align) TextOption {
	return func(opts *inode.TextOptions) {
		switch a {
		case AlignStart:
			opts.Align = inode.TextAlignLeft
		case AlignCenter:
			opts.Align = inode.TextAlignCenter
		case AlignEnd:
			opts.Align = inode.TextAlignRight
		}
	}
}

// WithTextStyle sets the style of a text node. Layout fields such as Width,
// Height, FlexGrow, and visual fields such as Foreground are supported.
func WithTextStyle(s Style) TextOption {
	return func(opts *inode.TextOptions) {
		opts.Style = s
	}
}

// View creates a container node with the given style and children.
func View(s Style, children ...Node) Node {
	return &inode.Node{
		Kind:     inode.ViewKind,
		Style:    s,
		Children: children,
	}
}

// ViewProps configures a View node with event handlers and focus.
type ViewProps struct {
	Style     Style
	OnKey     func(KeyEvent) bool
	Focusable bool
	AutoFocus bool
	Disabled  bool
}

// ViewWith creates a container node with props and children.
func ViewWith(props ViewProps, children ...Node) Node {
	return &inode.Node{
		Kind:      inode.ViewKind,
		Style:     props.Style,
		Children:  children,
		OnKey:     props.OnKey,
		Focusable: props.Focusable,
		AutoFocus: props.AutoFocus,
		Disabled:  props.Disabled,
	}
}

// Text creates a text leaf node.
func Text(value string, opts ...TextOption) Node {
	var textOpts inode.TextOptions
	for _, o := range opts {
		o(&textOpts)
	}
	return &inode.Node{
		Kind:     inode.TextKind,
		Text:     value,
		TextOpts: textOpts,
		Style:    textOpts.Style,
	}
}

// Keyed wraps a child node with an explicit reconciliation key.
func Keyed(key string, child Node) Node {
	return &inode.Node{
		Kind:     inode.KeyedKind,
		Key:      key,
		Children: []*inode.Node{child},
	}
}

// Component wraps a function component so it participates in the runtime.
// The function is called on every render; state is preserved across calls
// via UseState.
//
// Prefer named component functions. Inline closures may not preserve identity
// unless wrapped in Keyed.
func Component(fn func() Node) Node {
	id := reflect.ValueOf(fn).Pointer()
	return &inode.Node{
		Kind:   inode.ComponentKind,
		CompFn: fn,
		CompID: id,
	}
}

// UseState returns the current value of a state slot and a setter function.
// It must only be called from within a component function.
func UseState[T any](initial T) (T, func(T)) {
	return runtime.UseState(initial)
}

// UseFocused reports whether the current component's rendered subtree contains
// the currently focused node. It must only be called from within a component
// function.
func UseFocused() bool {
	return runtime.IsFocused()
}

// ButtonProps configures a Button widget.
type ButtonProps struct {
	Label        string
	OnPress      func()
	Style        Style
	FocusedStyle Style
	AutoFocus    bool
	Disabled     bool
}

func overlayVisualStyle(base, focus Style) Style {
	out := base
	if focus.Foreground.IsSpecified() {
		out.Foreground = focus.Foreground
	}
	if focus.Background.IsSpecified() {
		out.Background = focus.Background
	}
	if focus.Bold {
		out.Bold = true
	}
	if focus.Italic {
		out.Italic = true
	}
	if focus.Underline {
		out.Underline = true
	}
	return out
}

// Button creates a focusable button that responds to Enter and Space.
//
// If rendering buttons in a dynamic or reordered list, wrap each Button in Keyed
// so reconciliation preserves the intended identity.
func Button(props ButtonProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		style := props.Style
		if focused {
			style = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		return ViewWith(
			ViewProps{
				Style:     style,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if ev.Key == KeyEnter {
						if props.OnPress != nil {
							props.OnPress()
						}
						return true
					}
					if ev.Key == KeyRune && ev.Rune == ' ' {
						if props.OnPress != nil {
							props.OnPress()
						}
						return true
					}
					return false
				},
			},
			Text(props.Label, WithAlign(AlignCenter), WithTextStyle(Style{FlexGrow: 1})),
		)
	})
}

// TextInputProps configures a TextInput widget.
type TextInputProps struct {
	Value        string
	OnChange     func(string)
	Placeholder  string
	Style        Style
	FocusedStyle Style
	AutoFocus    bool
	Disabled     bool
}

// TextInput creates a focusable single-line text input.
// It is a controlled component: pass the current value through Value and receive
// edits through OnChange. The parent is responsible for updating state.
// TextInput is single-line. Long values are clipped by the view bounds;
// horizontal scrolling and cursor movement are not yet implemented.
func TextInput(props TextInputProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		displayValue := props.Value
		if displayValue == "" {
			displayValue = props.Placeholder
		}

		nd := ViewWith(
			ViewProps{
				Style:     s,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled {
						return false
					}
					switch ev.Key {
					case KeyRune:
						if ev.Rune == 0 {
							return false
						}
						if props.OnChange != nil {
							props.OnChange(props.Value + string(ev.Rune))
						}
						return true
					case KeyBackspace:
						runes := []rune(props.Value)
						if len(runes) == 0 {
							return false
						}
						if props.OnChange != nil {
							props.OnChange(string(runes[:len(runes)-1]))
						}
						return true
					}
					return false
				},
			},
			Text(displayValue, WithTextStyle(Style{FlexGrow: 1})),
		)

		if focused && !props.Disabled {
			nd.CursorVisible = true
			nd.CursorX = runewidth.StringWidth(props.Value)
			nd.CursorY = 0
		}

		return nd
	})
}

// Run initializes the terminal, runs the main loop, and cleans up on exit.
// The root function is called on every render to produce the new node tree.
func Run(root func() Node) error {
	h, err := host.NewTcellHost()
	if err != nil {
		return err
	}
	if err := h.Init(); err != nil {
		return err
	}
	defer h.Close()

	rt := runtime.New()
	w, h2 := h.Size()

	doRender := func(prev *screen.Buffer) *screen.Buffer {
		rootNode := root()
		rt.Update(rootNode)
		rt.RunLayout(w, h2)
		next := screen.NewBuffer(w, h2)
		rt.Render(next)
		render.FlushDiff(prev, next, h)
		if cx, cy, ok := rt.Cursor(); ok {
			h.ShowCursor(cx, cy)
		} else {
			h.HideCursor()
		}
		h.Show()
		return next
	}

	// Initial render. ensureFocus inside Update may set dirty if focusable
	// nodes were found, requiring a second render to reflect focus state.
	prev := doRender(nil)
	if rt.IsDirty() {
		prev = doRender(prev)
	}

	for {
		ev := h.PollEvent()
		switch ev.Kind {
		case event.QuitKind:
			return nil
		case event.ResizeKind:
			w, h2 = ev.Width, ev.Height
			rt.MarkDirty()
		case event.KeyKind:
			// Quit on Ctrl-C.
			if ev.Key.IsCtrlC() {
				return nil
			}
			rt.HandleEvent(ev)
		}

		if rt.IsDirty() {
			prev = doRender(prev)
		}
	}
}
