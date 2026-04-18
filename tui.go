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

// MouseButton identifies which mouse button or wheel direction was activated.
type MouseButton = input.MouseButton

const (
	MouseNone      = input.MouseNone
	MouseLeft      = input.MouseLeft
	MouseMiddle    = input.MouseMiddle
	MouseRight     = input.MouseRight
	MouseWheelUp   = input.MouseWheelUp
	MouseWheelDown = input.MouseWheelDown
)

// MouseEvent holds data for a mouse handler.
type MouseEvent = input.MousePress

// ViewProps configures a View node with event handlers and focus.
type ViewProps struct {
	Style     Style
	OnKey     func(KeyEvent) bool
	OnMouse   func(MouseEvent) bool
	Focusable bool
	AutoFocus bool
	Disabled  bool

	// FocusScope traps focus traversal within this view's subtree.
	FocusScope bool
}

// ViewWith creates a container node with props and children.
func ViewWith(props ViewProps, children ...Node) Node {
	return &inode.Node{
		Kind:       inode.ViewKind,
		Style:      props.Style,
		Children:   children,
		OnKey:      props.OnKey,
		OnMouse:    props.OnMouse,
		Focusable:  props.Focusable,
		AutoFocus:  props.AutoFocus,
		Disabled:   props.Disabled,
		FocusScope: props.FocusScope,
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

// Overlay stacks children on top of each other.
// Each child receives the same layout bounds. Later children render over earlier children.
func Overlay(children ...Node) Node {
	return OverlayWith(Style{}, children...)
}

// OverlayWith is like Overlay but with an explicit style.
func OverlayWith(style Style, children ...Node) Node {
	return &inode.Node{
		Kind:     inode.OverlayKind,
		Style:    style,
		Children: children,
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

// Router provides navigation state and actions.
type Router struct {
	Path    string
	CanBack bool
	Push    func(string)
	Replace func(string)
	Back    func() bool
}

// UseRouter returns a Router with the given initial path.
// It uses UseState to maintain a navigation stack.
func UseRouter(initial string) Router {
	stack, setStack := UseState([]string{initial})

	if len(stack) == 0 {
		stack = []string{initial}
	}

	current := stack[len(stack)-1]

	return Router{
		Path:    current,
		CanBack: len(stack) > 1,
		Push: func(path string) {
			next := append(append([]string{}, stack...), path)
			setStack(next)
		},
		Replace: func(path string) {
			next := append([]string{}, stack...)
			if len(next) == 0 {
				next = []string{path}
			} else {
				next[len(next)-1] = path
			}
			setStack(next)
		},
		Back: func() bool {
			if len(stack) <= 1 {
				return false
			}
			next := append([]string{}, stack[:len(stack)-1]...)
			setStack(next)
			return true
		},
	}
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
					if ev.Key == KeyEnter || (ev.Key == KeyRune && ev.Rune == ' ') {
						if props.OnPress == nil {
							return false
						}
						props.OnPress()
						return true
					}
					return false
				},
				OnMouse: func(ev MouseEvent) bool {
					if ev.Button&MouseLeft != 0 && !props.Disabled && props.OnPress != nil {
						props.OnPress()
						return true
					}
					return false
				},
			},
			Text(props.Label, WithAlign(AlignCenter), WithTextStyle(Style{FlexGrow: 1})),
		)
	})
}

// CheckboxProps configures a Checkbox widget.
type CheckboxProps struct {
	Label    string
	Value    bool
	OnChange func(bool)

	// Style is the base style for the checkbox's focusable view.
	Style Style

	// FocusedStyle overlays only visual attributes when the checkbox is focused.
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

// Checkbox creates a focusable checkbox that responds to Space and Enter.
// It is a controlled component: pass the current value through Value and receive
// toggles through OnChange. The parent is responsible for updating state.
func Checkbox(props CheckboxProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		mark := "[ ]"
		if props.Value {
			mark = "[x]"
		}

		return ViewWith(
			ViewProps{
				Style:     s,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled {
						return false
					}
					if ev.Key == KeyEnter || (ev.Key == KeyRune && ev.Rune == ' ') {
						if props.OnChange == nil {
							return false
						}
						props.OnChange(!props.Value)
						return true
					}
					return false
				},
			},
			Text(mark+" "+props.Label, WithTextStyle(Style{FlexGrow: 1})),
		)
	})
}

// DialogProps configures a Dialog widget.
type DialogProps struct {
	Style         Style
	BackdropStyle Style
	OnClose       func()
	CloseOnEsc    bool
}

// Dialog returns a full-screen focus scope that:
// 1. Draws a backdrop.
// 2. Centers the dialog child.
// 3. Calls OnClose when Esc is pressed and CloseOnEsc is true.
// 4. Prevents background focus traversal through focus scope behavior.
func Dialog(props DialogProps, child Node) Node {
	return Component(func() Node {
		return ViewWith(
			ViewProps{
				FocusScope: true,
				Style: Style{
					FlexGrow:   1,
					Direction:  Column,
					AlignItems: AlignCenter,
					Justify:    JustifyCenter,
					Background: props.BackdropStyle.Background,
					Foreground: props.BackdropStyle.Foreground,
				},
				OnKey: func(ev KeyEvent) bool {
					if props.CloseOnEsc && ev.Key == KeyEsc {
						if props.OnClose != nil {
							props.OnClose()
						}
						return true
					}
					return false
				},
			},
			View(props.Style, child),
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
						if props.OnChange == nil {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)-1)
						nextRunes = append(nextRunes, runes[:cursor-1]...)
						nextRunes = append(nextRunes, runes[cursor:]...)
						props.OnChange(string(nextRunes))
						setCursor(cursor - 1)
						return true
					case KeyDelete:
						if cursor >= len(runes) {
							return false
						}
						if props.OnChange == nil {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)-1)
						nextRunes = append(nextRunes, runes[:cursor]...)
						nextRunes = append(nextRunes, runes[cursor+1:]...)
						props.OnChange(string(nextRunes))
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
						if props.OnChange == nil {
							return false
						}
						nextRunes := make([]rune, 0, len(runes)+1)
						nextRunes = append(nextRunes, runes[:cursor]...)
						nextRunes = append(nextRunes, ev.Rune)
						nextRunes = append(nextRunes, runes[cursor:]...)
						props.OnChange(string(nextRunes))
						setCursor(cursor + 1)
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

// ListProps configures a List widget.
type ListProps struct {
	Items         []string
	SelectedIndex int
	OnSelect      func(int)

	Style        Style
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

// List creates a focusable vertical list where items are navigated with Up/Down keys.
// It is a controlled component: it displays Items and calls OnSelect with the
// next index. The parent must update SelectedIndex for the visual state to change.
func List(props ListProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		s.Direction = Column

		items := make([]Node, len(props.Items))
		for i, label := range props.Items {
			prefix := "  "
			if i == props.SelectedIndex {
				prefix = "> "
			}

			itemStyle := Style{}
			if focused && i == props.SelectedIndex {
				itemStyle = overlayVisualStyle(Style{}, props.FocusedStyle)
			}

			items[i] = View(itemStyle, Text(prefix+label))
		}

		return ViewWith(
			ViewProps{
				Style:     s,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled || props.OnSelect == nil || len(props.Items) == 0 {
						return false
					}

					selected := clampInt(props.SelectedIndex, 0, len(props.Items)-1)

					switch ev.Key {
					case KeyUp:
						if selected > 0 {
							props.OnSelect(selected - 1)
							return true
						}
					case KeyDown:
						if selected < len(props.Items)-1 {
							props.OnSelect(selected + 1)
							return true
						}
					}
					return false
				},
			},
			items...,
		)
	})
}

// RadioOption configures a single RadioGroup option.
type RadioOption struct {
	Label string
	Value string
}

// RadioGroupProps configures a RadioGroup widget.
type RadioGroupProps struct {
	Options  []RadioOption
	Value    string
	OnChange func(string)

	Style        Style
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

func radioSelectedIndex(options []RadioOption, value string) int {
	for i, opt := range options {
		if opt.Value == value {
			return i
		}
	}
	return -1
}

// RadioGroup creates a focusable vertical radio group navigated with Up/Down.
// It is controlled: it displays Value and calls OnChange with the next option
// value. The parent must update Value for the visual state to change.
func RadioGroup(props RadioGroupProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		s.Direction = Column

		selected := radioSelectedIndex(props.Options, props.Value)

		items := make([]Node, len(props.Options))
		for i, opt := range props.Options {
			mark := "( )"
			if i == selected {
				mark = "(*)"
			}

			itemStyle := Style{}
			if focused && i == selected {
				itemStyle = overlayVisualStyle(Style{}, props.FocusedStyle)
			}

			items[i] = View(itemStyle, Text(mark+" "+opt.Label))
		}

		selectIndex := func(i int) bool {
			if i < 0 || i >= len(props.Options) {
				return false
			}
			if props.OnChange == nil {
				return false
			}
			next := props.Options[i].Value
			if next == props.Value {
				return true
			}
			props.OnChange(next)
			return true
		}

		return ViewWith(
			ViewProps{
				Style:     s,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled || len(props.Options) == 0 {
						return false
					}

					switch ev.Key {
					case KeyUp:
						if props.OnChange == nil {
							return false
						}
						if selected < 0 {
							return selectIndex(len(props.Options) - 1)
						}
						if selected == 0 {
							return false
						}
						return selectIndex(selected - 1)

					case KeyDown:
						if props.OnChange == nil {
							return false
						}
						if selected < 0 {
							return selectIndex(0)
						}
						if selected >= len(props.Options)-1 {
							return false
						}
						return selectIndex(selected + 1)

					case KeyHome:
						if selected == 0 {
							return false
						}
						return selectIndex(0)

					case KeyEnd:
						if selected == len(props.Options)-1 {
							return false
						}
						return selectIndex(len(props.Options) - 1)

					case KeyEnter:
						if selected < 0 {
							return selectIndex(0)
						}
						return selectIndex(selected)

					case KeyRune:
						if ev.Rune == ' ' {
							if selected < 0 {
								return selectIndex(0)
							}
							return selectIndex(selected)
						}
					}

					return false
				},
			},
			items...,
		)
	})
}

// SelectProps configures a Select widget.
type SelectProps struct {
	Options  []RadioOption
	Value    string
	OnChange func(string)

	Style        Style
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

// Select creates a focusable widget that cycles through options.
// It is controlled: it displays the label of the option matching Value and calls
// OnChange with the next option value when pressed or navigated.
func Select(props SelectProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		selected := radioSelectedIndex(props.Options, props.Value)
		display := ""
		if selected >= 0 {
			display = props.Options[selected].Label
		}

		selectIndex := func(i int) bool {
			if props.OnChange == nil || len(props.Options) == 0 {
				return false
			}
			if i < 0 {
				i = len(props.Options) - 1
			}
			if i >= len(props.Options) {
				i = 0
			}

			next := props.Options[i].Value
			if next == props.Value {
				return true
			}

			props.OnChange(next)
			return true
		}

		nextIndex := func() int {
			if selected < 0 {
				return 0
			}
			return selected + 1
		}

		prevIndex := func() int {
			if selected < 0 {
				return len(props.Options) - 1
			}
			return selected - 1
		}

		return ViewWith(
			ViewProps{
				Style:     s,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled || len(props.Options) == 0 || props.OnChange == nil {
						return false
					}

					switch ev.Key {
					case KeyLeft, KeyUp:
						return selectIndex(prevIndex())
					case KeyRight, KeyDown, KeyEnter:
						return selectIndex(nextIndex())
					case KeyHome:
						if selected == 0 {
							return false
						}
						return selectIndex(0)
					case KeyEnd:
						if selected == len(props.Options)-1 {
							return false
						}
						return selectIndex(len(props.Options) - 1)
					case KeyRune:
						if ev.Rune == ' ' {
							return selectIndex(nextIndex())
						}
					}

					return false
				},
			},
			Text(" < "+display+" > ", WithTextStyle(Style{FlexGrow: 1})),
		)
	})
}

// TextPanelProps configures a scrollable read-only text panel.
type TextPanelProps struct {
	Text string

	// Style is the base style for the panel.
	Style Style

	// FocusedStyle overlays only visual attributes when the panel is focused:
	// Foreground, Background, Bold, Italic, and Underline.
	// Layout fields such as Width, Height, Padding, Gap, Border, and FlexGrow
	// are intentionally ignored.
	FocusedStyle Style

	// WordWrap wraps long lines to the panel content width.
	// When false, long lines are horizontally clipped and Left/Right scroll.
	WordWrap bool

	// ShowScrollbar draws a vertical scrollbar when the visual text content
	// has more lines than the visible viewport.
	ShowScrollbar bool

	AutoFocus bool
	Disabled  bool
}

// TextPanel creates a focusable scrollable read-only text panel.
//
// Keyboard behavior:
//   - Up/Down scroll one visual line.
//   - PgUp/PgDown scroll one viewport page.
//   - Home/End scroll to top/bottom.
//   - Left/Right scroll horizontally only when WordWrap is false.
//
// TextPanel is read-only. It does not call OnChange and does not edit text.
func TextPanel(props TextPanelProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		return &inode.Node{
			Kind:      inode.TextPanelKind,
			Text:      props.Text,
			Style:     s,
			Focusable: !props.Disabled,
			AutoFocus: props.AutoFocus,
			Disabled:  props.Disabled,
			TextPanelOpts: inode.TextPanelOptions{
				WordWrap:      props.WordWrap,
				ShowScrollbar: props.ShowScrollbar,
			},
		}
	})
}

// Run initializes the terminal, runs the main loop, and cleans up on exit.
// The root function is called on every render to produce the new node tree.
type hostFactory func() (host.Host, error)

// Run starts the TUI event loop, rendering root on each update until Ctrl-C or
// the terminal closes.
func Run(root func() Node) error {
	return runWithHost(root, func() (host.Host, error) {
		h, err := host.NewTcellHost()
		if err != nil {
			return nil, err
		}
		return h, nil
	})
}

// runWithHost is the testable core of Run. It accepts a host factory so tests
// can inject a fake host without requiring a real terminal.
func runWithHost(root func() Node, newHost hostFactory) error {
	h, err := newHost()
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

	// Render repeatedly until stable, so ensureFocus dirty cycles settle.
	renderUntilClean := func(prev *screen.Buffer) *screen.Buffer {
		for {
			prev = doRender(prev)
			if !rt.IsDirty() {
				return prev
			}
		}
	}

	prev := renderUntilClean(nil)

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
		case event.MouseKind:
			rt.HandleEvent(ev)
		}

		if rt.IsDirty() {
			prev = renderUntilClean(prev)
		}
	}
}
