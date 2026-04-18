// Package ansi parses a small subset of ANSI SGR styling into rich text spans.
package ansi

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/smason/earlgray/internal/color"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/style"
)

// ParseText converts ANSI SGR sequences into styled text spans.
func ParseText(value string) []node.TextSpan {
	var spans []node.TextSpan
	var text strings.Builder
	current := style.Style{}

	flush := func() {
		if text.Len() == 0 {
			return
		}
		segment := text.String()
		text.Reset()
		if len(spans) > 0 && spans[len(spans)-1].Style == current {
			spans[len(spans)-1].Text += segment
			return
		}
		spans = append(spans, node.TextSpan{
			Text:  segment,
			Style: current,
		})
	}

	for i := 0; i < len(value); {
		if value[i] != 0x1b || i+1 >= len(value) || value[i+1] != '[' {
			text.WriteByte(value[i])
			i++
			continue
		}

		j := i + 2
		for j < len(value) && !unicode.IsLetter(rune(value[j])) {
			j++
		}
		if j >= len(value) {
			text.WriteString(value[i:])
			break
		}

		final := value[j]
		if final != 'm' {
			i = j + 1
			continue
		}

		flush()
		current = applySGR(current, value[i+2:j])
		i = j + 1
	}

	flush()
	return spans
}

func applySGR(current style.Style, params string) style.Style {
	if params == "" {
		return style.Style{}
	}

	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		code := 0
		if parts[i] != "" {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				continue
			}
			code = n
		}

		switch {
		case code == 0:
			current = style.Style{}
		case code == 1:
			current.Bold = true
		case code == 2:
			current.Faint = true
		case code == 3:
			current.Italic = true
		case code == 4:
			current.Underline = true
		case code == 7:
			current.Reverse = true
		case code == 9:
			current.Strikethrough = true
		case code == 22:
			current.Bold = false
			current.Faint = false
		case code == 23:
			current.Italic = false
		case code == 24:
			current.Underline = false
		case code == 27:
			current.Reverse = false
		case code == 29:
			current.Strikethrough = false
		case code == 39:
			current.Foreground = color.Color{}
		case code == 49:
			current.Background = color.Color{}
		case 30 <= code && code <= 37:
			current.Foreground = color.ANSIColor(code - 30)
		case 40 <= code && code <= 47:
			current.Background = color.ANSIColor(code - 40)
		case 90 <= code && code <= 97:
			current.Foreground = color.ANSIColor(code - 90 + 8)
		case 100 <= code && code <= 107:
			current.Background = color.ANSIColor(code - 100 + 8)
		case code == 38 || code == 48:
			if i+2 < len(parts) && parts[i+1] == "5" {
				n, err := strconv.Atoi(parts[i+2])
				if err == nil {
					if code == 38 {
						current.Foreground = color.ANSIColor(n)
					} else {
						current.Background = color.ANSIColor(n)
					}
				}
				i += 2
			}
		}
	}

	return current
}
