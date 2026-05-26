package tui

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/smasonuk/earlgray/internal/ansi"
	"github.com/smasonuk/earlgray/internal/event"
	"github.com/smasonuk/earlgray/internal/host"
	"github.com/smasonuk/earlgray/internal/input"
	inode "github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/render"
	"github.com/smasonuk/earlgray/internal/runtime"
	"github.com/smasonuk/earlgray/internal/screen"
)

// Node is the opaque tree element returned by View, Text, Keyed, etc.
type Node = *inode.Node

// Key identifies a special key or KeyRune for printable characters.
type Key = input.Key

const (
	KeyUnknown   = input.KeyUnknown
	KeyRune      = input.KeyRune
	KeyCtrlC     = input.KeyCtrlC
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

// Fragment groups children without adding visual styling.
func Fragment(children ...Node) Node {
	return View(Style{}, children...)
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

// MouseAction identifies the type of mouse activity.
type MouseAction = input.MouseAction

const (
	MousePress   = input.ActionPress
	MouseRelease = input.ActionRelease
	MouseMotion  = input.ActionMotion
)

// MouseEvent holds data for a mouse handler.
type MouseEvent = input.MousePress

// ViewProps configures a View node with event handlers and focus.
type ViewProps struct {
	Style        Style
	OnKey        func(KeyEvent) bool
	OnKeyCapture func(KeyEvent) bool
	OnMouse      func(MouseEvent) bool
	OnPaste      func(string) bool
	Focusable    bool
	AutoFocus    bool
	Disabled     bool

	// FocusScope traps focus traversal within this view's subtree.
	FocusScope bool
}

// ViewWith creates a container node with props and children.
func ViewWith(props ViewProps, children ...Node) Node {
	return &inode.Node{
		Kind:         inode.ViewKind,
		Style:        props.Style,
		Children:     children,
		OnKey:        props.OnKey,
		OnKeyCapture: props.OnKeyCapture,
		OnMouse:      props.OnMouse,
		OnPaste:      props.OnPaste,
		Focusable:    props.Focusable,
		AutoFocus:    props.AutoFocus,
		Disabled:     props.Disabled,
		FocusScope:   props.FocusScope,
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

// TextSpan is a styled segment in a RichText node.
type TextSpan struct {
	Text  string
	Style Style
}

func toInternalTextSpans(spans []TextSpan) []inode.TextSpan {
	if len(spans) == 0 {
		return nil
	}
	out := make([]inode.TextSpan, len(spans))
	for i, span := range spans {
		out[i] = inode.TextSpan{
			Text:  span.Text,
			Style: span.Style,
		}
	}
	return out
}

// RichText creates a text node with multiple styled spans.
func RichText(spans ...TextSpan) Node {
	return &inode.Node{
		Kind:  inode.RichTextKind,
		Spans: toInternalTextSpans(spans),
	}
}

// ANSIText parses ANSI SGR styling sequences into styled spans.
func ANSIText(value string, opts ...TextOption) Node {
	var textOpts inode.TextOptions
	for _, o := range opts {
		o(&textOpts)
	}
	return &inode.Node{
		Kind:     inode.RichTextKind,
		Spans:    ansi.ParseText(value),
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

// ComponentWithKey wraps a function component with a reconciliation key.
// Use this for inline components whose state must be preserved across renders.
func ComponentWithKey(key string, fn func() Node) Node {
	return Keyed(key, Component(fn))
}

// UseState returns the current value of a state slot and a setter function.
// It must only be called from within a component function.
func UseState[T any](initial T) (T, func(T)) {
	return runtime.UseState(initial)
}

// UseStateWithUpdater returns the current value of a state slot, a setter function,
// and a functional updater.
// It must only be called from within a component function.
func UseStateWithUpdater[T any](initial T) (T, func(T), func(func(T) T)) {
	return runtime.UseStateWithUpdater(initial)
}

// UseReducer returns the current state and a dispatch function to apply actions.
// It must only be called from within a component function.
func UseReducer[S any, A any](reducer func(S, A) S, initial S) (S, func(A)) {
	return runtime.UseReducer(reducer, initial)
}

// UseRef returns a stable pointer for component-local mutable state that does
// not itself trigger rerenders when mutated.
//
// It must only be called from within a component function.
func UseRef[T any](initial T) *T {
	return runtime.UseRef(initial)
}

// UseEffect registers a component-local side effect.
//
// The effect runs after the rendered tree has been committed. If the dependency
// values are unchanged since the previous render, the effect is not rerun.
//
// If the effect returns a cleanup function, the cleanup runs:
//   - before the effect reruns due to dependency changes
//   - when the component unmounts
//
// With zero dependencies, the effect runs once on mount.
func UseEffect(effect func() func(), deps ...any) {
	runtime.UseEffect(effect, deps...)
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

var DefaultSpinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼",
	"⠴", "⠦", "⠧", "⠇", "⠏",
}

const DefaultSpinnerInterval = 100 * time.Millisecond

// SpinnerProps configures the built-in spinner widget.
type SpinnerProps struct {
	Frames   []string
	Label    string
	Style    Style
	Active   bool
	Interval time.Duration
}

func spinnerFrames(props SpinnerProps) []string {
	if len(props.Frames) > 0 {
		return props.Frames
	}
	return DefaultSpinnerFrames
}

func spinnerFramesKey(frames []string) string {
	var b strings.Builder
	for _, frame := range frames {
		b.WriteString(strconv.Itoa(len(frame)))
		b.WriteRune(':')
		b.WriteString(frame)
		b.WriteRune(';')
	}
	return b.String()
}

// Spinner renders a text spinner that can animate while active.
func Spinner(props SpinnerProps) Node {
	return Component(func() Node {
		frames := spinnerFrames(props)
		frameCount := len(frames)

		frameIndex, _, setFrameIndexGuarded := runtime.UseStateGuarded(0)
		generation := runtime.UseRef(0)

		interval := props.Interval
		if interval <= 0 {
			interval = DefaultSpinnerInterval
		}

		active := props.Active
		framesKey := spinnerFramesKey(frames)

		UseEffect(func() func() {
			if !active || frameCount <= 1 {
				return nil
			}

			stop := make(chan struct{})
			var stopped atomic.Bool
			(*generation)++
			gen := *generation
			i := 0
			if frameCount > 0 {
				i = frameIndex % frameCount
			}

			go func() {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						if stopped.Load() {
							return
						}
						next := (i + 1) % frameCount
						i = next
						setFrameIndexGuarded(next, func() bool {
							return *generation == gen
						})
					case <-stop:
						return
					}
				}
			}()

			return func() {
				stopped.Store(true)
				(*generation)++
				close(stop)
			}
		}, active, interval, framesKey)

		displayIndex := 0
		if frameCount > 0 {
			displayIndex = frameIndex % frameCount
		}

		frame := ""
		if frameCount > 0 {
			frame = frames[displayIndex]
		}

		text := frame
		if props.Label != "" {
			if text == "" {
				text = props.Label
			} else {
				text += " " + props.Label
			}
		}

		return Text(text, WithTextStyle(props.Style))
	})
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
	if focus.Faint {
		out.Faint = true
	}
	if focus.Strikethrough {
		out.Strikethrough = true
	}
	if focus.Reverse {
		out.Reverse = true
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
					if ev.Button&MouseLeft != 0 && ev.Action == MousePress && !props.Disabled && props.OnPress != nil {
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
				OnMouse: func(ev MouseEvent) bool {
					if ev.Button&MouseLeft == 0 || ev.Action != MousePress || props.Disabled || props.OnChange == nil {
						return false
					}
					props.OnChange(!props.Value)
					return true
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

	// PlaceholderStyle overlays visual attributes when Value is empty and the
	// placeholder text is shown. Empty defaults to a muted gray foreground.
	PlaceholderStyle Style

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

func textInputPlaceholderStyle(base Style, placeholder Style) Style {
	if placeholder.Foreground.IsSpecified() || placeholder.Background.IsSpecified() ||
		placeholder.Bold || placeholder.Italic || placeholder.Underline ||
		placeholder.Faint || placeholder.Strikethrough || placeholder.Reverse {
		return overlayVisualStyle(base, placeholder)
	}
	return overlayVisualStyle(base, Style{Foreground: ANSIColor(8)})
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
		placeholderVisible := displayValue == ""
		if placeholderVisible {
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
		textStyle := Style{FlexGrow: 1}
		if placeholderVisible {
			textStyle = textInputPlaceholderStyle(textStyle, props.PlaceholderStyle)
		}

		nd := ViewWith(
			ViewProps{
				Style: s,
				OnMouse: func(ev MouseEvent) bool {
					if props.Disabled || ev.Button&MouseLeft == 0 || ev.Action != MousePress {
						return false
					}

					runes := []rune(props.Value)
					cw, fixed := textInputContentWidthFromStyle(props.Style)

					start := 0
					if fixed && focused {
						maxWidth := cw - 1
						start = cursor
						wBefore := 0
						for start > 0 {
							rw := runewidth.RuneWidth(runes[start-1])
							if wBefore+rw > maxWidth {
								break
							}
							wBefore += rw
							start--
						}
					}

					x := 0
					for i := start; i < len(runes); i++ {
						rw := runewidth.RuneWidth(runes[i])
						if ev.LocalX < x+rw {
							// Clicked on this rune.
							if ev.LocalX < x+(rw+1)/2 {
								setCursor(i)
							} else {
								setCursor(i + 1)
							}
							return true
						}
						x += rw
					}
					setCursor(len(runes))
					return true
				},
				OnPaste: func(text string) bool {
					if props.Disabled || props.OnChange == nil {
						return false
					}
					pasted := sanitizeTextInputPaste(text)
					if pasted == "" {
						return false
					}
					pastedRunes := []rune(pasted)
					nextRunes := make([]rune, 0, len(runes)+len(pastedRunes))
					nextRunes = append(nextRunes, runes[:cursor]...)
					nextRunes = append(nextRunes, pastedRunes...)
					nextRunes = append(nextRunes, runes[cursor:]...)
					props.OnChange(string(nextRunes))
					setCursor(cursor + len(pastedRunes))
					return true
				},
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
			Text(displayValue, WithTextStyle(textStyle)),
		)

		if focused && !props.Disabled {
			nd.CursorVisible = true
			nd.CursorX = cursorX
			nd.CursorY = 0
		}

		return nd
	})
}

func sanitizeTextInputPaste(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

// TextAreaProps configures a TextArea widget.
type TextAreaProps struct {
	Value       string
	OnChange    func(string)
	OnSubmit    func(string)
	Placeholder string

	// Style is the base style for the textarea's focusable view.
	Style Style

	// FocusedStyle overlays only visual attributes when the textarea is focused:
	// Foreground, Background, Bold, Italic, and Underline.
	// Layout fields such as Width, Height, Padding, Gap, Border, and FlexGrow
	// are intentionally ignored.
	FocusedStyle Style

	// NoWordWrap disables word wrapping. When true, long lines scroll horizontally.
	// By default TextArea word-wraps.
	NoWordWrap bool

	// ShowScrollbar draws a vertical scrollbar when content exceeds the viewport.
	ShowScrollbar bool

	// SubmitOnCtrlEnter calls OnSubmit on Ctrl+Enter.
	// Plain Enter always inserts a newline.
	SubmitOnCtrlEnter bool

	// OnCopy is called with the selected text when the user presses Ctrl+C
	// and there is a non-empty selection. Use RunWithOptions{DisableCtrlCQuit: true}
	// so that Ctrl+C is delivered to the textarea rather than quitting.
	OnCopy func(string)

	AutoFocus bool
	Disabled  bool
	ReadOnly  bool
}

// TextArea creates a focusable editable multi-line text input.
//
// TextArea is a controlled component: pass Value and receive edits through
// OnChange. The parent is responsible for updating state.
//
// Keyboard behavior:
//   - Rune keys insert text.
//   - Enter inserts a newline.
//   - Ctrl+Enter calls OnSubmit when SubmitOnCtrlEnter is true.
//   - Backspace/Delete remove text.
//   - Left/Right move by rune.
//   - Up/Down move by visual line.
//   - PgUp/PgDown move by viewport page.
//   - Home/End move to start/end of the current visual line.
//
// By default TextArea word-wraps. Set NoWordWrap to enable horizontal scrolling.
func TextArea(props TextAreaProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		s := props.Style
		if focused {
			s = overlayVisualStyle(props.Style, props.FocusedStyle)
		}

		return &inode.Node{
			Kind:      inode.TextAreaKind,
			Text:      props.Value,
			Style:     s,
			Focusable: !props.Disabled,
			AutoFocus: props.AutoFocus,
			Disabled:  props.Disabled,
			TextAreaOpts: inode.TextAreaOptions{
				Placeholder:       props.Placeholder,
				WordWrap:          !props.NoWordWrap,
				ShowScrollbar:     props.ShowScrollbar,
				OnChange:          props.OnChange,
				OnSubmit:          props.OnSubmit,
				SubmitOnCtrlEnter: props.SubmitOnCtrlEnter,
				OnCopy:            props.OnCopy,
				ReadOnly:          props.ReadOnly,
			},
		}
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

			idx := i
			items[i] = ViewWith(
				ViewProps{
					Style: itemStyle,
					OnMouse: func(ev MouseEvent) bool {
						if ev.Button&MouseLeft == 0 || ev.Action != MousePress || props.Disabled || props.OnSelect == nil {
							return false
						}
						props.OnSelect(idx)
						return true
					},
				},
				Text(prefix+label),
			)
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

			idx := i
			items[i] = ViewWith(
				ViewProps{
					Style: itemStyle,
					OnMouse: func(ev MouseEvent) bool {
						if ev.Button&MouseLeft == 0 || ev.Action != MousePress || props.Disabled {
							return false
						}
						return selectIndex(idx)
					},
				},
				Text(mark+" "+opt.Label),
			)
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

// SideTab configures a single SideTabs tab.
type SideTab struct {
	Label    string
	Value    string
	Content  Node
	Disabled bool
}

// SideTabsProps configures a SideTabs widget.
//
// Value is the active tab value. OnChange receives the next tab value when the
// user selects a different enabled tab. Disabled disables the whole component;
// SideTab.Disabled disables only that tab.
type SideTabsProps struct {
	Tabs     []SideTab
	Value    string
	OnChange func(string)

	Style Style

	TabListStyle     Style
	TabStyle         Style
	ActiveTabStyle   Style
	FocusedTabStyle  Style
	DisabledTabStyle Style

	PanelStyle Style

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
				OnMouse: func(ev MouseEvent) bool {
					if ev.Button&MouseLeft == 0 || ev.Action != MousePress || props.Disabled || len(props.Options) == 0 || props.OnChange == nil {
						return false
					}
					return selectIndex(nextIndex())
				},
			},
			Text(" < "+display+" > ", WithTextStyle(Style{FlexGrow: 1})),
		)
	})
}

func sideTabsSelectedIndex(tabs []SideTab, value string) int {
	for i, tab := range tabs {
		if tab.Value == value {
			return i
		}
	}
	return -1
}

func sideTabsFirstEnabledIndex(tabs []SideTab) int {
	for i, tab := range tabs {
		if !tab.Disabled {
			return i
		}
	}
	return -1
}

func sideTabsLastEnabledIndex(tabs []SideTab) int {
	for i := len(tabs) - 1; i >= 0; i-- {
		if !tabs[i].Disabled {
			return i
		}
	}
	return -1
}

func sideTabsNextEnabledIndex(tabs []SideTab, from int) int {
	for i := from + 1; i < len(tabs); i++ {
		if !tabs[i].Disabled {
			return i
		}
	}
	return -1
}

func sideTabsPrevEnabledIndex(tabs []SideTab, from int) int {
	for i := from - 1; i >= 0; i-- {
		if !tabs[i].Disabled {
			return i
		}
	}
	return -1
}

// SideTabs creates a focusable vertical tab list on the left and renders the
// active tab's content in a panel on the right.
//
// It is controlled: pass the active tab value through Value and receive
// changes through OnChange. The parent must update Value for the active panel
// to change.
//
// Keyboard behavior:
//   - Up/Down select the previous/next enabled tab.
//   - Home/End select the first/last enabled tab.
//   - Tab/Shift+Tab move focus out of the tab list.
//
// Only the active tab's Content is rendered. Inactive tab content is unmounted;
// hoist state above SideTabs if tab-local state must persist across tab changes.
func SideTabs(props SideTabsProps) Node {
	return Component(func() Node {
		focused := UseFocused()

		active := sideTabsSelectedIndex(props.Tabs, props.Value)
		if active < 0 {
			active = sideTabsFirstEnabledIndex(props.Tabs)
		}

		selectIndex := func(i int) bool {
			if props.Disabled || props.OnChange == nil || i < 0 || i >= len(props.Tabs) || props.Tabs[i].Disabled {
				return false
			}
			next := props.Tabs[i].Value
			if next == props.Value {
				return true
			}
			props.OnChange(next)
			return true
		}

		tabRows := make([]Node, len(props.Tabs))
		for i, tab := range props.Tabs {
			prefix := "  "
			if i == active {
				prefix = "> "
			}

			tabStyle := props.TabStyle
			if i == active {
				tabStyle = overlayVisualStyle(tabStyle, props.ActiveTabStyle)
			}
			if focused && i == active {
				tabStyle = overlayVisualStyle(tabStyle, props.FocusedTabStyle)
			}
			if props.Disabled || tab.Disabled {
				tabStyle = overlayVisualStyle(tabStyle, props.DisabledTabStyle)
			}

			idx := i
			tabRows[i] = ViewWith(
				ViewProps{
					Style: tabStyle,
					OnMouse: func(ev MouseEvent) bool {
						if ev.Button&MouseLeft == 0 || ev.Action != MousePress || props.Disabled || props.OnChange == nil {
							return false
						}
						if props.Tabs[idx].Disabled {
							return false
						}
						if idx == active {
							return true
						}
						return selectIndex(idx)
					},
				},
				Text(prefix+tab.Label),
			)
		}

		tabListStyle := props.TabListStyle
		tabListStyle.Direction = Column
		tabList := ViewWith(
			ViewProps{
				Style:     tabListStyle,
				Focusable: !props.Disabled,
				AutoFocus: props.AutoFocus,
				Disabled:  props.Disabled,
				OnKey: func(ev KeyEvent) bool {
					if props.Disabled || props.OnChange == nil || len(props.Tabs) == 0 {
						return false
					}

					switch ev.Key {
					case KeyUp:
						return selectIndex(sideTabsPrevEnabledIndex(props.Tabs, active))
					case KeyDown:
						return selectIndex(sideTabsNextEnabledIndex(props.Tabs, active))
					case KeyHome:
						first := sideTabsFirstEnabledIndex(props.Tabs)
						if active == first {
							return false
						}
						return selectIndex(first)
					case KeyEnd:
						last := sideTabsLastEnabledIndex(props.Tabs)
						if active == last {
							return false
						}
						return selectIndex(last)
					}

					return false
				},
			},
			tabRows...,
		)

		panelStyle := props.PanelStyle
		if panelStyle.FlexGrow == 0 && panelStyle.Width.Kind == DimAuto {
			panelStyle.FlexGrow = 1
		}

		panelChildren := []Node{}
		if active >= 0 && active < len(props.Tabs) && props.Tabs[active].Content != nil {
			panelChildren = append(panelChildren, props.Tabs[active].Content)
		}
		panel := View(panelStyle, panelChildren...)

		outerStyle := props.Style
		if outerStyle.FlexGrow == 0 {
			outerStyle.FlexGrow = 1
		}
		outerStyle.Direction = Row

		return View(
			outerStyle,
			tabList,
			panel,
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

	// AutoScrollBottom forces the panel to remain pinned to the bottom.
	AutoScrollBottom bool

	// ResetScrollKey resets the retained scroll position when its value changes.
	ResetScrollKey string

	// InitialScrollX and InitialScrollY apply on mount and when ResetScrollKey changes.
	InitialScrollX int
	InitialScrollY int

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
				WordWrap:         props.WordWrap,
				ShowScrollbar:    props.ShowScrollbar,
				AutoScrollBottom: props.AutoScrollBottom,
				ResetScrollKey:   props.ResetScrollKey,
				InitialScrollX:   props.InitialScrollX,
				InitialScrollY:   props.InitialScrollY,
			},
		}
	})
}

// AppHandle exposes safe app-level actions to background goroutines.
type AppHandle struct {
	Post  func(func())
	Quit  func()
	Every func(time.Duration, func()) func()
}

// AppContext provides app-level actions to components.
type AppContext = runtime.AppContext

// UseApp returns the current AppContext.
// It must only be called from within a component function.
func UseApp() AppContext {
	return runtime.UseApp()
}

// UseChannel registers a subscription to a channel.
// It starts a goroutine that reads values from the channel and executes
// onValue for each received value on the EarlGray app loop.
// The subscription is stopped when the component unmounts or dependencies change.
// If ch is nil, no subscription is created.
func UseChannel[T any](ch <-chan T, onValue func(T), deps ...any) {
	appCtx := UseApp()

	UseEffect(func() func() {
		if ch == nil {
			return nil
		}

		stop := make(chan struct{})
		var stopped atomic.Bool

		go func() {
			for {
				select {
				case <-stop:
					return
				case val, ok := <-ch:
					if !ok {
						return
					}
					if stopped.Load() {
						return
					}
					if onValue != nil {
						appCtx.Post(func() {
							onValue(val)
						})
					}
				}
			}
		}()

		return func() {
			stopped.Store(true)
			close(stop)
		}
	}, deps...)
}

// RunOptions configures the TUI event loop.
type RunOptions struct {
	// DisableCtrlCQuit prevents EarlGray from automatically quitting on Ctrl-C.
	// When true, Ctrl-C is delivered to app key handlers as KeyCtrlC.
	DisableCtrlCQuit bool
	OnStart          func(AppHandle)
}

// Run initializes the terminal, runs the main loop, and cleans up on exit.
// The root function is called on every render to produce the new node tree.
type hostFactory func() (host.Host, error)

// Run starts the TUI event loop, rendering root on each update until Ctrl-C or
// the terminal closes.
func Run(root func() Node) error {
	return RunWithOptions(root, RunOptions{})
}

// RunWithOptions starts the TUI event loop with configurable startup and quit behavior.
func RunWithOptions(root func() Node, opts RunOptions) error {
	return runWithHost(root, opts, func() (host.Host, error) {
		h, err := host.NewTcellHost()
		if err != nil {
			return nil, err
		}
		return h, nil
	})
}

// runWithHost is the testable core of Run. It accepts a host factory so tests
// can inject a fake host without requiring a real terminal.
func runWithHost(root func() Node, opts RunOptions, newHost hostFactory) error {
	h, err := newHost()
	if err != nil {
		return err
	}
	if err := h.Init(); err != nil {
		return err
	}
	defer h.Close()

	appEvents := make(chan func(), 1024)
	hostEvents := make(chan event.Event, 64)
	quit := make(chan struct{})
	done := make(chan struct{})

	var quitOnce sync.Once
	var doneOnce sync.Once
	var shuttingDown atomic.Bool

	shutdown := func() {
		shuttingDown.Store(true)
		quitOnce.Do(func() {
			close(quit)
		})
		doneOnce.Do(func() {
			close(done)
		})
	}
	defer shutdown()

	var handle AppHandle
	handle = AppHandle{
		Post: func(fn func()) {
			if fn == nil {
				return
			}
			if shuttingDown.Load() {
				return
			}
			select {
			case <-quit:
				return
			default:
			}
			select {
			case appEvents <- fn:
			case <-quit:
			}
		},
		Quit: shutdown,
		Every: func(d time.Duration, fn func()) func() {
			if d <= 0 || fn == nil {
				return func() {}
			}

			stop := make(chan struct{})
			var stopOnce sync.Once

			go func() {
				ticker := time.NewTicker(d)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						handle.Post(fn)
					case <-stop:
						return
					case <-quit:
						return
					}
				}
			}()

			return func() {
				stopOnce.Do(func() {
					close(stop)
				})
			}
		},
	}

	rt := runtime.New()
	rt.SetAppContext(AppContext{
		Post:  handle.Post,
		Quit:  handle.Quit,
		Every: handle.Every,
	})
	defer rt.Dispose()
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
		rt.RunEffects()
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

	drainAppEvents := func() {
		for {
			select {
			case fn := <-appEvents:
				if fn != nil {
					fn()
					rt.MarkDirty()
				}
			default:
				return
			}
		}
	}

	prev := renderUntilClean(nil)

	go func() {
		for {
			ev := h.PollEvent()
			select {
			case hostEvents <- ev:
			case <-done:
				return
			}
			if ev.Kind == event.QuitKind {
				return
			}
		}
	}()

	if opts.OnStart != nil {
		go opts.OnStart(handle)
	}

	for {
		select {
		case <-quit:
			return nil
		case fn := <-appEvents:
			if fn != nil {
				fn()
				rt.MarkDirty()
			}
		case ev := <-hostEvents:
			switch ev.Kind {
			case event.QuitKind:
				shutdown()
				return nil
			case event.ResizeKind:
				w, h2 = ev.Width, ev.Height
				rt.MarkDirty()
			case event.KeyKind:
				if !opts.DisableCtrlCQuit && ev.Key.IsCtrlC() {
					shutdown()
					return nil
				}
				if rt.HandleEvent(ev) {
					rt.MarkDirty()
				}
			case event.PasteKind:
				if rt.HandleEvent(ev) {
					rt.MarkDirty()
				}
			case event.MouseKind:
				if rt.HandleEvent(ev) {
					rt.MarkDirty()
				}
			}
		}

		drainAppEvents()

		if rt.IsDirty() {
			prev = renderUntilClean(prev)
		}
	}
}

// ScrollableListItem configures a single ScrollableList row.
type ScrollableListItem struct {
	ID    string
	Label string
}

// ScrollableListProps configures a bounded, selectable vertical list.
type ScrollableListProps struct {
	Items []ScrollableListItem

	// Controlled selected index. The parent owns the selected item.
	SelectedIndex int

	// Called when the user selects, navigates to, or activates an item.
	OnSelect func(int)

	// Optional activation callback for Enter.
	// If nil, Enter behaves like OnSelect(selectedIndex), if OnSelect exists.
	OnActivate func(int)

	// Optional click callback for mouse activation.
	// If nil, click behaves like OnSelect(clickedIndex), preserving List-style selection.
	OnClick func(int)

	// Optional fallback row count used when layout height is otherwise unconstrained.
	// If <= 0, default to 8.
	VisibleRows int

	// Text displayed when Items is empty.
	EmptyText string

	// ShowFooter renders a footer such as "showing 1-8 of 30" when clipped.
	ShowFooter bool

	Style        Style
	FocusedStyle Style

	AutoFocus bool
	Disabled  bool
}

// ScrollableList creates a focusable bounded list with keyboard, wheel, and
// click selection. It renders only the rows that fit in its allocated viewport.
func ScrollableList(props ScrollableListProps) Node {
	items := make([]inode.ScrollableListItem, len(props.Items))
	for i, item := range props.Items {
		items[i] = inode.ScrollableListItem{
			ID:    item.ID,
			Label: item.Label,
		}
	}

	return &inode.Node{
		Kind:      inode.ScrollableListKind,
		Style:     props.Style,
		Focusable: !props.Disabled,
		AutoFocus: props.AutoFocus,
		Disabled:  props.Disabled,
		ScrollableListOpts: inode.ScrollableListOptions{
			Items:         items,
			SelectedIndex: props.SelectedIndex,
			OnSelect:      props.OnSelect,
			OnActivate:    props.OnActivate,
			OnClick:       props.OnClick,
			VisibleRows:   props.VisibleRows,
			EmptyText:     props.EmptyText,
			ShowFooter:    props.ShowFooter,
			FocusedStyle:  props.FocusedStyle,
		},
	}
}
