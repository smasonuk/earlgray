import sys

with open("tui.go", "r") as f:
    content = f.read()

# Replace ButtonProps
button_props_old = """// ButtonProps configures a Button widget.
type ButtonProps struct {
	Label        string
	OnPress      func()
	Style        Style
	FocusedStyle Style
	AutoFocus    bool
	Disabled     bool
}"""

button_props_new = """// ButtonProps configures a Button widget.
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
}"""

content = content.replace(button_props_old, button_props_new)

# Replace TextInputProps and TextInput
text_input_old_start = "// TextInputProps configures a TextInput widget."
run_func_start = "// Run initializes the terminal"
idx1 = content.find(text_input_old_start)
idx2 = content.find(run_func_start)
if idx1 == -1 or idx2 == -1:
    print("Could not find TextInput bounds")
    sys.exit(1)

text_input_new = """// TextInputProps configures a TextInput widget.
type TextInputProps struct {
	Value        string
	OnChange     func(string)
	OnSubmit     func(string)
	Placeholder  string

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
// TextInput is single-line. Long values are clipped by the view bounds;
// horizontal scrolling and cursor movement are not yet implemented.
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

"""

content = content[:idx1] + text_input_new + content[idx2:]

with open("tui.go", "w") as f:
    f.write(content)
