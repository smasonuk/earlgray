package textflow

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// WrapLines wraps the given text to the specified width using word-based wrapping.
func WrapLines(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	var visual []string
	logicalLines := strings.Split(text, "\n")
	for _, line := range logicalLines {
		if line == "" {
			visual = append(visual, "")
			continue
		}

		words := strings.Split(line, " ")
		var currentLine string
		var currentWidth int

		for i, word := range words {
			wordWidth := runewidth.StringWidth(word)

			// If it's not the first word, try to add a space.
			if i > 0 {
				if currentWidth+1+wordWidth <= width {
					currentLine += " " + word
					currentWidth += 1 + wordWidth
					continue
				} else {
					// Cannot fit space + word, so finish currentLine and start new.
					if currentLine != "" {
						visual = append(visual, currentLine)
					}
					currentLine = ""
					currentWidth = 0
				}
			}

			// Current word might be longer than width.
			for runewidth.StringWidth(word) > width {
				// Hard break the word.
				idx := 0
				w := 0
				for _, r := range word {
					rw := runewidth.RuneWidth(r)
					if w+rw > width {
						break
					}
					w += rw
					idx += len(string(r))
				}

				// If width is so small it can't fit a single rune, force one.
				if idx == 0 && len(word) > 0 {
					_, size := utf8.DecodeRuneInString(word)
					idx = size
				}

				visual = append(visual, word[:idx])
				word = word[idx:]
			}

			// Now we have the remainder of the word which fits in width.
			// But it might not fit in currentLine.
			wordWidth = runewidth.StringWidth(word)
			if currentWidth+wordWidth > width {
				if currentLine != "" {
					visual = append(visual, currentLine)
				}
				currentLine = word
				currentWidth = wordWidth
			} else {
				currentLine += word
				currentWidth += wordWidth
			}
		}
		if currentLine != "" {
			visual = append(visual, currentLine)
		}
	}
	return visual
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
