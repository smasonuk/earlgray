package runtime

import (
	"github.com/mattn/go-runewidth"
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/screen"
	"github.com/smason/earlgray/internal/style"
)

type textAreaVisualLine struct {
	Text       string
	Start, End int // rune indexes in the underlying value; End excludes newline
}

func resetTextAreaState(inst *Instance) {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextAreaKind {
		return
	}
	inst.scrollX = 0
	inst.scrollY = 0
	inst.textAreaCursor = len([]rune(inst.nd.Text))
}

func textAreaDisplayText(value string, opts node.TextAreaOptions) string {
	if value == "" && opts.Placeholder != "" {
		return opts.Placeholder
	}
	return value
}

func textAreaVisualLines(text string, wordWrap bool, width int) []textAreaVisualLine {
	runes := []rune(text)
	var lines []textAreaVisualLine

	start := 0
	for start <= len(runes) {
		end := start
		for end < len(runes) && runes[end] != '\n' {
			end++
		}

		appendTextAreaLogicalLine(&lines, runes, start, end, wordWrap, width)

		if end == len(runes) {
			break
		}
		start = end + 1
	}

	if len(lines) == 0 {
		lines = append(lines, textAreaVisualLine{Start: 0, End: 0})
	}
	return lines
}

func appendTextAreaLogicalLine(lines *[]textAreaVisualLine, runes []rune, start, end int, wordWrap bool, width int) {
	if start == end {
		*lines = append(*lines, textAreaVisualLine{
			Text:  "",
			Start: start,
			End:   end,
		})
		return
	}

	if !wordWrap || width <= 0 {
		*lines = append(*lines, textAreaVisualLine{
			Text:  string(runes[start:end]),
			Start: start,
			End:   end,
		})
		return
	}

	pos := start
	for pos < end {
		w := 0
		i := pos
		lastWordBreak := -1
		seenNonSpace := false

		for i < end {
			r := runes[i]
			rw := runewidth.RuneWidth(r)
			if rw == 0 {
				i++
				continue
			}

			if r == ' ' && seenNonSpace {
				lastWordBreak = i
			}
			if w+rw > width {
				break
			}
			if r != ' ' {
				seenNonSpace = true
			}

			w += rw
			i++
		}

		if i >= end {
			*lines = append(*lines, textAreaVisualLine{
				Text:  string(runes[pos:end]),
				Start: pos,
				End:   end,
			})
			break
		}

		if lastWordBreak > pos {
			lineEnd := lastWordBreak
			for lineEnd > pos && runes[lineEnd-1] == ' ' {
				lineEnd--
			}

			*lines = append(*lines, textAreaVisualLine{
				Text:  string(runes[pos:lineEnd]),
				Start: pos,
				End:   lineEnd,
			})

			pos = lastWordBreak + 1
			for pos < end && runes[pos] == ' ' {
				pos++
			}
			continue
		}

		if i == pos {
			i = pos + 1
		}

		*lines = append(*lines, textAreaVisualLine{
			Text:  string(runes[pos:i]),
			Start: pos,
			End:   i,
		})
		pos = i
	}
}

func textAreaMaxLineWidth(lines []textAreaVisualLine) int {
	maxW := 0
	for _, line := range lines {
		if w := runewidth.StringWidth(line.Text); w > maxW {
			maxW = w
		}
	}
	return maxW
}

func textAreaViewportWidth(text string, opts node.TextAreaOptions, contentW, contentH int) int {
	viewportW := contentW
	if viewportW <= 0 {
		return 0
	}

	lines := textAreaVisualLines(text, opts.WordWrap, viewportW)
	overflowY := len(lines) > contentH

	if opts.ShowScrollbar && overflowY && contentW > 1 {
		viewportW = contentW - 1
	}
	return viewportW
}

func textAreaCursorLineAndX(lines []textAreaVisualLine, runes []rune, cursor int) (int, int) {
	if len(lines) == 0 {
		return 0, 0
	}

	cursor = clampIntRuntime(cursor, 0, len(runes))

	for i, line := range lines {
		// Prefer the next visual line at wrap boundaries.
		if cursor == line.Start {
			return i, 0
		}
		if cursor > line.Start && cursor <= line.End {
			return i, runewidth.StringWidth(string(runes[line.Start:cursor]))
		}
		if cursor < line.Start {
			prev := i - 1
			if prev < 0 {
				return 0, 0
			}
			return prev, runewidth.StringWidth(lines[prev].Text)
		}
	}

	last := len(lines) - 1
	line := lines[last]
	x := runewidth.StringWidth(line.Text)
	if cursor >= line.Start && cursor <= line.End {
		x = runewidth.StringWidth(string(runes[line.Start:cursor]))
	}
	return last, x
}

func textAreaCursorIndexAt(lines []textAreaVisualLine, runes []rune, row, x int) int {
	if len(lines) == 0 {
		return 0
	}
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		return len(runes)
	}

	line := lines[row]
	col := 0
	for i := line.Start; i < line.End; i++ {
		rw := runewidth.RuneWidth(runes[i])
		if rw == 0 {
			continue
		}
		if x < col+rw {
			if x < col+(rw+1)/2 {
				return i
			}
			return i + 1
		}
		col += rw
	}
	return line.End
}

func textAreaEnsureCursorVisible(inst *Instance, lines []textAreaVisualLine, runes []rune, viewportW int) {
	content := inst.layout.Content
	if content.W <= 0 || content.H <= 0 || viewportW <= 0 {
		return
	}

	opts := inst.nd.TextAreaOpts
	row, x := textAreaCursorLineAndX(lines, runes, inst.textAreaCursor)

	if row < inst.scrollY {
		inst.scrollY = row
	}
	if row >= inst.scrollY+content.H {
		inst.scrollY = row - content.H + 1
	}

	maxY := len(lines) - content.H
	if maxY < 0 {
		maxY = 0
	}
	inst.scrollY = clampIntRuntime(inst.scrollY, 0, maxY)

	if opts.WordWrap {
		inst.scrollX = 0
		return
	}

	if x < inst.scrollX {
		inst.scrollX = x
	}
	if x >= inst.scrollX+viewportW {
		inst.scrollX = x - viewportW + 1
	}

	maxX := textAreaMaxLineWidth(lines) - viewportW
	if maxX < 0 {
		maxX = 0
	}
	inst.scrollX = clampIntRuntime(inst.scrollX, 0, maxX)
}

func handleTextAreaKey(inst *Instance, press input.KeyPress) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextAreaKind {
		return false
	}

	opts := inst.nd.TextAreaOpts
	value := inst.nd.Text
	runes := []rune(value)
	inst.textAreaCursor = clampIntRuntime(inst.textAreaCursor, 0, len(runes))
	cursor := inst.textAreaCursor

	content := inst.layout.Content
	viewportW := textAreaViewportWidth(value, opts, content.W, content.H)
	if viewportW <= 0 {
		viewportW = 1
	}
	lines := textAreaVisualLines(value, opts.WordWrap, viewportW)

	setCursor := func(next int) bool {
		next = clampIntRuntime(next, 0, len(runes))
		if next == inst.textAreaCursor {
			return false
		}
		inst.textAreaCursor = next
		textAreaEnsureCursorVisible(inst, lines, runes, viewportW)
		return true
	}

	replace := func(nextRunes []rune, nextCursor int) bool {
		if opts.OnChange == nil {
			return false
		}
		opts.OnChange(string(nextRunes))
		inst.textAreaCursor = clampIntRuntime(nextCursor, 0, len(nextRunes))
		return true
	}

	switch press.Key {
	case input.KeyLeft:
		return setCursor(cursor - 1)

	case input.KeyRight:
		return setCursor(cursor + 1)

	case input.KeyUp:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		if row <= 0 {
			return false
		}
		return setCursor(textAreaCursorIndexAt(lines, runes, row-1, x))

	case input.KeyDown:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		if row >= len(lines)-1 {
			return false
		}
		return setCursor(textAreaCursorIndexAt(lines, runes, row+1, x))

	case input.KeyPgUp:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		nextRow := row - content.H
		if nextRow < 0 {
			nextRow = 0
		}
		return setCursor(textAreaCursorIndexAt(lines, runes, nextRow, x))

	case input.KeyPgDown:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		nextRow := row + content.H
		if nextRow >= len(lines) {
			nextRow = len(lines) - 1
		}
		return setCursor(textAreaCursorIndexAt(lines, runes, nextRow, x))

	case input.KeyHome:
		row, _ := textAreaCursorLineAndX(lines, runes, cursor)
		return setCursor(lines[row].Start)

	case input.KeyEnd:
		row, _ := textAreaCursorLineAndX(lines, runes, cursor)
		return setCursor(lines[row].End)

	case input.KeyBackspace:
		if cursor == 0 {
			return false
		}
		next := make([]rune, 0, len(runes)-1)
		next = append(next, runes[:cursor-1]...)
		next = append(next, runes[cursor:]...)
		return replace(next, cursor-1)

	case input.KeyDelete:
		if cursor >= len(runes) {
			return false
		}
		next := make([]rune, 0, len(runes)-1)
		next = append(next, runes[:cursor]...)
		next = append(next, runes[cursor+1:]...)
		return replace(next, cursor)

	case input.KeyEnter:
		if opts.SubmitOnCtrlEnter && press.Mod&input.ModCtrl != 0 {
			if opts.OnSubmit == nil {
				return false
			}
			opts.OnSubmit(value)
			return true
		}

		next := make([]rune, 0, len(runes)+1)
		next = append(next, runes[:cursor]...)
		next = append(next, '\n')
		next = append(next, runes[cursor:]...)
		return replace(next, cursor+1)

	case input.KeyRune:
		if press.Rune == 0 {
			return false
		}
		next := make([]rune, 0, len(runes)+1)
		next = append(next, runes[:cursor]...)
		next = append(next, press.Rune)
		next = append(next, runes[cursor:]...)
		return replace(next, cursor+1)
	}

	return false
}

func handleTextAreaClick(inst *Instance, localX, localY int) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextAreaKind {
		return false
	}

	value := inst.nd.Text
	runes := []rune(value)
	opts := inst.nd.TextAreaOpts
	content := inst.layout.Content

	viewportW := textAreaViewportWidth(value, opts, content.W, content.H)
	if viewportW <= 0 {
		return false
	}

	if localX >= viewportW {
		localX = viewportW - 1
	}
	if localX < 0 {
		localX = 0
	}
	if localY < 0 {
		localY = 0
	}

	lines := textAreaVisualLines(value, opts.WordWrap, viewportW)
	row := inst.scrollY + localY
	x := localX
	if !opts.WordWrap {
		x += inst.scrollX
	}

	inst.textAreaCursor = textAreaCursorIndexAt(lines, runes, row, x)
	textAreaEnsureCursorVisible(inst, lines, runes, viewportW)
	return true
}

func scrollTextArea(inst *Instance, delta int) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextAreaKind {
		return false
	}

	content := inst.layout.Content
	if content.W <= 0 || content.H <= 0 {
		return false
	}

	opts := inst.nd.TextAreaOpts
	displayText := textAreaDisplayText(inst.nd.Text, opts)
	viewportW := textAreaViewportWidth(displayText, opts, content.W, content.H)
	lines := textAreaVisualLines(displayText, opts.WordWrap, viewportW)

	maxY := len(lines) - content.H
	if maxY < 0 {
		maxY = 0
	}

	oldY := inst.scrollY
	inst.scrollY = clampIntRuntime(inst.scrollY+delta, 0, maxY)
	return inst.scrollY != oldY
}

func renderTextArea(inst *Instance, buf *screen.Buffer, content style.Rect, s style.Style, cursor *cursorState) {
	if content.W <= 0 || content.H <= 0 {
		return
	}

	opts := inst.nd.TextAreaOpts
	value := inst.nd.Text
	displayText := textAreaDisplayText(value, opts)

	viewportW := content.W
	lines := textAreaVisualLines(displayText, opts.WordWrap, viewportW)

	overflowY := len(lines) > content.H
	showScrollbar := opts.ShowScrollbar && overflowY && content.W > 1
	if showScrollbar {
		viewportW = content.W - 1
		lines = textAreaVisualLines(displayText, opts.WordWrap, viewportW)
		overflowY = len(lines) > content.H
		showScrollbar = opts.ShowScrollbar && overflowY && content.W > 1
	}

	maxY := len(lines) - content.H
	if maxY < 0 {
		maxY = 0
	}
	inst.scrollY = clampIntRuntime(inst.scrollY, 0, maxY)

	maxX := 0
	if !opts.WordWrap {
		maxX = textAreaMaxLineWidth(lines) - viewportW
		if maxX < 0 {
			maxX = 0
		}
	}
	inst.scrollX = clampIntRuntime(inst.scrollX, 0, maxX)

	valueRunes := []rune(value)
	cursorLines := textAreaVisualLines(value, opts.WordWrap, viewportW)
	inst.textAreaCursor = clampIntRuntime(inst.textAreaCursor, 0, len(valueRunes))
	textAreaEnsureCursorVisible(inst, cursorLines, valueRunes, viewportW)

	textStyle := screenCellStyleFromStyle(s)

	for row := 0; row < content.H; row++ {
		lineIdx := inst.scrollY + row
		if lineIdx >= len(lines) {
			break
		}

		line := lines[lineIdx]
		y := content.Y + row

		if opts.WordWrap {
			buf.DrawTextClipped(content.X, y, line.Text, textStyle, content.X, content.Y, viewportW, content.H)
		} else {
			buf.DrawTextClipped(content.X-inst.scrollX, y, line.Text, textStyle, content.X, content.Y, viewportW, content.H)
		}
	}

	if showScrollbar {
		drawTextPanelScrollbar(buf, content, len(lines), content.H, inst.scrollY, s)
	}

	if cursor != nil && inst.runtime != nil && inst.runtime.focused == inst && !inst.nd.Disabled {
		row, x := textAreaCursorLineAndX(cursorLines, valueRunes, inst.textAreaCursor)

		cy := row - inst.scrollY
		cx := x
		if !opts.WordWrap {
			cx -= inst.scrollX
		}

		if cy >= 0 && cy < content.H && viewportW > 0 {
			cx = clampIntRuntime(cx, 0, viewportW-1)

			cursor.visible = true
			cursor.x = content.X + cx
			cursor.y = content.Y + cy
		}
	}
}
