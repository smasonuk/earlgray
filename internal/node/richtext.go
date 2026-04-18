package node

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// SplitTextSpansLines breaks rich text spans into visual lines at newline boundaries.
func SplitTextSpansLines(spans []TextSpan) [][]TextSpan {
	lines := [][]TextSpan{{}}
	for _, span := range spans {
		parts := strings.Split(span.Text, "\n")
		for i, part := range parts {
			if part != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], TextSpan{
					Text:  part,
					Style: span.Style,
				})
			}
			if i < len(parts)-1 {
				lines = append(lines, []TextSpan{})
			}
		}
	}
	if len(lines) == 0 {
		return [][]TextSpan{{}}
	}
	return lines
}

// RichTextLineWidth returns the display width of a rich text line.
func RichTextLineWidth(line []TextSpan) int {
	width := 0
	for _, span := range line {
		width += runewidth.StringWidth(span.Text)
	}
	return width
}
