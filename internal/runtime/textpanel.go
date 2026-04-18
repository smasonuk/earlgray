package runtime

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/style"
	"github.com/smason/earlgray/internal/screen"
)

func handleTextPanelKey(inst *Instance, press input.KeyPress) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextPanelKind {
		return false
	}

	content := inst.layout.Content
	if content.W <= 0 || content.H <= 0 {
		return false
	}

	opts := inst.nd.TextPanelOpts
	viewportW := textPanelViewportWidth(inst.nd.Text, opts, content.W, content.H)
	visualLines := textPanelVisualLines(inst.nd.Text, opts.WordWrap, viewportW)

	maxY := len(visualLines) - content.H
	if maxY < 0 {
		maxY = 0
	}

	maxX := 0
	if !opts.WordWrap {
		maxX = textPanelMaxLineWidth(visualLines) - viewportW
		if maxX < 0 {
			maxX = 0
		}
	}

	oldX, oldY := inst.scrollX, inst.scrollY

	switch press.Key {
	case input.KeyUp:
		inst.scrollY--
	case input.KeyDown:
		inst.scrollY++
	case input.KeyPgUp:
		inst.scrollY -= content.H
	case input.KeyPgDown:
		inst.scrollY += content.H
	case input.KeyHome:
		inst.scrollY = 0
		inst.scrollX = 0
	case input.KeyEnd:
		inst.scrollY = maxY
	case input.KeyLeft:
		if opts.WordWrap {
			return false
		}
		inst.scrollX--
	case input.KeyRight:
		if opts.WordWrap {
			return false
		}
		inst.scrollX++
	default:
		return false
	}

	inst.scrollY = clampIntRuntime(inst.scrollY, 0, maxY)
	inst.scrollX = clampIntRuntime(inst.scrollX, 0, maxX)

	return inst.scrollX != oldX || inst.scrollY != oldY
}

func clampIntRuntime(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func textPanelVisualLines(text string, wordWrap bool, width int) []string {
	if wordWrap {
		return wrapTextPanelLines(text, width)
	}
	return strings.Split(text, "\n")
}

func wrapTextPanelLines(text string, width int) []string {
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

			if i > 0 {
				if currentWidth+1+wordWidth <= width {
					currentLine += " " + word
					currentWidth += 1 + wordWidth
					continue
				} else {
					visual = append(visual, currentLine)
					currentLine = ""
					currentWidth = 0
				}
			}

			for runewidth.StringWidth(word) > width {
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
				if idx == 0 && len(word) > 0 {
					_, size := utf8.DecodeRuneInString(word)
					idx = size
				}
				visual = append(visual, word[:idx])
				word = word[idx:]
			}
			currentLine = word
			currentWidth = runewidth.StringWidth(word)
		}
		visual = append(visual, currentLine)
	}
	return visual
}

func textPanelMaxLineWidth(lines []string) int {
	max := 0
	for _, l := range lines {
		w := runewidth.StringWidth(l)
		if w > max {
			max = w
		}
	}
	return max
}

func textPanelViewportWidth(text string, opts node.TextPanelOptions, contentW, contentH int) int {
	viewportW := contentW
	visualLines := textPanelVisualLines(text, opts.WordWrap, viewportW)
	overflowY := len(visualLines) > contentH
	if opts.ShowScrollbar && overflowY && contentW > 1 {
		viewportW = contentW - 1
	}
	return viewportW
}

const (
	textPanelScrollbarTrack = '│'
	textPanelScrollbarThumb = '█'
)

func drawTextPanelScrollbar(buf *screen.Buffer, content style.Rect, totalLines, viewportH, scrollY int, s style.Style) {
	if viewportH <= 0 || totalLines <= viewportH {
		return
	}

	x := content.X + content.W - 1
	trackH := viewportH
	thumbH := (viewportH * viewportH) / totalLines
	if thumbH < 1 {
		thumbH = 1
	}
	if thumbH > trackH {
		thumbH = trackH
	}

	maxScroll := totalLines - viewportH
	thumbTop := 0
	if maxScroll > 0 {
		thumbTop = (scrollY * (trackH - thumbH)) / maxScroll
	}

	trackStyle := screen.CellStyle{Fg: s.Foreground, Bg: s.Background}
	thumbStyle := screen.CellStyle{Fg: s.Foreground, Bg: s.Background}

	for i := 0; i < trackH; i++ {
		if i >= thumbTop && i < thumbTop+thumbH {
			buf.SetCell(x, content.Y+i, textPanelScrollbarThumb, thumbStyle)
		} else {
			buf.SetCell(x, content.Y+i, textPanelScrollbarTrack, trackStyle)
		}
	}
}

func renderTextPanel(inst *Instance, buf *screen.Buffer, content style.Rect, s style.Style) {
	if content.W <= 0 || content.H <= 0 {
		return
	}

	opts := inst.nd.TextPanelOpts

	viewportW := content.W
	visualLines := textPanelVisualLines(inst.nd.Text, opts.WordWrap, viewportW)

	overflowY := len(visualLines) > content.H
	showScrollbar := opts.ShowScrollbar && overflowY && content.W > 1

	if showScrollbar {
		viewportW = content.W - 1
		visualLines = textPanelVisualLines(inst.nd.Text, opts.WordWrap, viewportW)
		overflowY = len(visualLines) > content.H
		showScrollbar = opts.ShowScrollbar && overflowY && content.W > 1
	}

	maxY := len(visualLines) - content.H
	if maxY < 0 {
		maxY = 0
	}
	inst.scrollY = clampIntRuntime(inst.scrollY, 0, maxY)

	maxX := 0
	if !opts.WordWrap {
		maxX = textPanelMaxLineWidth(visualLines) - viewportW
		if maxX < 0 {
			maxX = 0
		}
	}
	inst.scrollX = clampIntRuntime(inst.scrollX, 0, maxX)

	textStyle := screen.CellStyle{
		Fg:        s.Foreground,
		Bg:        s.Background,
		Bold:      s.Bold,
		Italic:    s.Italic,
		Underline: s.Underline,
	}

	for row := 0; row < content.H; row++ {
		lineIdx := inst.scrollY + row
		if lineIdx >= len(visualLines) {
			break
		}

		line := visualLines[lineIdx]
		y := content.Y + row

		if opts.WordWrap {
			buf.DrawTextClipped(content.X, y, line, textStyle, content.X, content.Y, viewportW, content.H)
		} else {
			// Draw the full line shifted left by scrollX. DrawTextClipped already
			// avoids partial wide-rune rendering at the clip boundaries.
			buf.DrawTextClipped(content.X-inst.scrollX, y, line, textStyle, content.X, content.Y, viewportW, content.H)
		}
	}

	if showScrollbar {
		drawTextPanelScrollbar(buf, content, len(visualLines), content.H, inst.scrollY, s)
	}
}
