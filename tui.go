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
	Label   string
	OnPress func()

	// Style is the base style for the button's focusable view.
	Style Style

	// FocusedStyle overlays only visual attributes when the button is focused:
	// Foreground, Background, Bold, Italic, and Underline.
	// Layout fields such as Width, Height, Padding, Gap, Border, and FlexGrow
	// are intentionally ignored.
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
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
	Value       string
	OnChange    func(string)
	OnSubmit    func(string)
	Placeholder string

	// Style is the base style for the input's focusable view.
	Style Style

	// FocusedStyle overlays only visual attributes when the input is focused:
	// Foreground, Background, Bold, Italic, and Underline.
	// Layout fields such as Width, Height, Padding, Gap, Border, and FlexGrow
	// are intentionally ignored.
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func textInputContentWidthFromStyle(s Style) (int, bool) {
	if s.Width.Kind != DimCells {
		return 0, false
	}

	w := s.Width.Value

	border := s.Border.Insets()
	w -= border.Left + border.Right
	w -= s.Padding.Left + s.Padding.Right

	if w < 0 {
		w = 0
	}
	return w, true
}

func textInputVisibleValue(value string, cursorRuneIndex int, contentWidth int, focused bool) (visible string, cursorX int) {
	if contentWidth <= 0 {
		return "", 0
	}
	if !focused {
		return value, 0
	}

	runes := []rune(value)
	cursorRuneIndex = clampInt(cursorRuneIndex, 0, len(runes))

	maxWidth := contentWidth - 1
	if maxWidth < 0 {
		maxWidth = 0
	}

	start := cursorRuneIndex
	wBefore := 0
	for start > 0 {
		rw := runewidth.RuneWidth(runes[start-1])
		if wBefore+rw > maxWidth {
			break
		}
		wBefore += rw
		start--
	}

	end := cursorRuneIndex
	wAfter := 0
	for end < len(runes) {
		rw := runewidth.RuneWidth(runes[end])
		if wBefore+wAfter+rw > maxWidth {
			break
		}
		wAfter += rw
		end++
	}

	vis := string(runes[start:end])
	// append one trailing space while focused, as current code does.
	vis += " "
	cursorX = runewidth.StringWidth(string(runes[start:cursorRuneIndex]))

	return vis, cursorX
}

// TextInput creates a focusable single-line text input.
// It is a controlled component: pass the current value through Value and receive
// edits through OnChange. The parent is responsible for updating state.
//
// TextInput is single-line. It supports cursor movement, insertion,
// Backspace, Delete, and Enter submission. Fixed-width inputs scroll
// horizontally to keep the cursor visible. Auto/flex-sized inputs still
// rely on normal clipping behavior.
func TextInput(props TextInputProps) Node {
	return Component(func() Node {
		focused := UseFocused()
		cursor, setCursor := UseState(len([]rune(props.Value)))

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		runes := []rune(props.Value)
		cursor = clampInt(cursor, 0, len(runes))

		displayValue := props.Value
		if displayValue == "" {
			displayValue = props.Placeholder
		}

		var cursorX int
		if cw, ok := textInputContentWidthFromStyle(props.Style); ok {
			displayValue, cursorX = textInputVisibleValue(displayValue, cursor, cw, focused && !props.Disabled)
		} else {
			if focused && !props.Disabled {
				displayValue += " "
			}
			cursorX = runewidth.StringWidth(string(runes[:cursor]))
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
					case KeyLeft:
						if cursor == 0 {
							return false
						}
						setCursor(cursor - 1)
						return true
					case KeyRight:
						if cursor >= len(runes) {
							return false
						}
						setCursor(cursor + 1)
						return true
					case KeyHome:
						if cursor == 0 {
							return false
						}
						setCursor(0)
						return true
					case KeyEnd:
						if cursor == len(runes) {
							return false
						}
						setCursor(len(runes))
						return true
					case KeyBackspace:
						if cursor == 0 {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)-1)
						nextRunes = append(nextRunes, runes[:cursor-1]...)
						nextRunes = append(nextRunes, runes[cursor:]...)
						if props.OnChange != nil {
							props.OnChange(string(nextRunes))
							setCursor(cursor - 1)
						}
						return true
					case KeyDelete:
						if cursor >= len(runes) {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)-1)
						nextRunes = append(nextRunes, runes[:cursor]...)
						nextRunes = append(nextRunes, runes[cursor+1:]...)
						if props.OnChange != nil {
							props.OnChange(string(nextRunes))
						}
						return true
					case KeyEnter:
						if props.OnSubmit == nil {
							return false
						}
						props.OnSubmit(props.Value)
						return true
					case KeyRune:
						if ev.Rune == 0 {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)+1)
						nextRunes = append(nextRunes, runes[:cursor]...)
						nextRunes = append(nextRunes, ev.Rune)
						nextRunes = append(nextRunes, runes[cursor:]...)
						if props.OnChange != nil {
							props.OnChange(string(nextRunes))
							setCursor(cursor + 1)
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
			nd.CursorX = cursorX
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
