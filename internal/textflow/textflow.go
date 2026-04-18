package textflow

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// WrapLines wraps the given text to the specified display width.
// It prefers breaking at spaces after non-space content (word-wrap), falling
// back to hard breaks when no word boundary is available. Wide runes are never
// split. Blank lines and leading spaces are preserved.
//
// When breaking at a word boundary the space itself is consumed and trailing
// spaces are trimmed from the emitted line; the next line starts at the first
// non-space character after the break.
func WrapLines(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	var visual []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			visual = append(visual, "")
			continue
		}
		visual = append(visual, wrapSingleLine([]rune(line), width)...)
	}
	return visual
}

// wrapSingleLine wraps a single non-empty logical line into visual lines.
func wrapSingleLine(runes []rune, width int) []string {
	n := len(runes)
	var result []string
	start := 0

	for start < n {
		w := 0
		end := start
		lastWordBreak := -1 // index of the last space that follows non-space content
		seenNonSpace := false

		for end < n {
			r := runes[end]
			rw := runewidth.RuneWidth(r)
			if rw == 0 {
				end++
				continue
			}
			// Record a word-break opportunity at spaces that follow non-space content.
			// We record it before the width check so the space at exactly position
			// `width` (display-wise) is still captured as a break point.
			if r == ' ' && seenNonSpace {
				lastWordBreak = end
			}
			if w+rw > width {
				break
			}
			if r != ' ' {
				seenNonSpace = true
			}
			w += rw
			end++
		}

		if end >= n {
			// Remaining text fits entirely on one line.
			result = append(result, string(runes[start:]))
			break
		}

		if lastWordBreak > start {
			// Break at the rightmost word boundary that fits.
			lineEnd := lastWordBreak
			// Trim trailing spaces from the emitted segment.
			for lineEnd > start && runes[lineEnd-1] == ' ' {
				lineEnd--
			}
			result = append(result, string(runes[start:lineEnd]))
			// Skip the break space and any consecutive spaces that follow.
			start = lastWordBreak + 1
			for start < n && runes[start] == ' ' {
				start++
			}
		} else {
			// No word boundary available: hard break at the display-width limit.
			if end == start {
				// Width is too small even for one rune; force a single rune.
				end = start + 1
			}
			result = append(result, string(runes[start:end]))
			start = end
		}
	}

	return result
}

// VisualLines returns the visual lines of the text, wrapping if wordWrap is true.
func VisualLines(text string, wordWrap bool, width int) []string {
	if wordWrap && width > 0 {
		return WrapLines(text, width)
	}
	return strings.Split(text, "\n")
}

// MaxLineWidth returns the maximum width of the given lines in terminal cells.
func MaxLineWidth(lines []string) int {
	maxW := 0
	for _, line := range lines {
		w := runewidth.StringWidth(line)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}
