package runtime

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/smasonuk/earlgray/internal/input"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/screen"
	"github.com/smasonuk/earlgray/internal/style"
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
	inst.textAreaSelectionAnchor = -1
	inst.textAreaDragging = false
	inst.textAreaEnsureCursorVisible = true
}

func textAreaClearSelection(inst *Instance) {
	inst.textAreaSelectionAnchor = -1
}

// textAreaSelectionRange returns the normalized [start, end) selection range.
// Returns ok=false if there is no anchor or the selection is empty.
func textAreaSelectionRange(inst *Instance, runeCount int) (start int, end int, ok bool) {
	if inst.textAreaSelectionAnchor < 0 {
		return 0, 0, false
	}
	anchor := clampIntRuntime(inst.textAreaSelectionAnchor, 0, runeCount)
	cursor := clampIntRuntime(inst.textAreaCursor, 0, runeCount)
	if anchor == cursor {
		return 0, 0, false
	}
	if anchor < cursor {
		return anchor, cursor, true
	}
	return cursor, anchor, true
}

// textAreaSelectedText returns the selected text if there is a non-empty selection.
func textAreaSelectedText(inst *Instance, runes []rune) (string, bool) {
	start, end, ok := textAreaSelectionRange(inst, len(runes))
	if !ok {
		return "", false
	}
	return string(runes[start:end]), true
}

// textAreaDeleteSelectionRunes deletes the selected range from runes.
// Returns the new rune slice, the cursor position for subsequent insertion, and whether a deletion occurred.
func textAreaDeleteSelectionRunes(inst *Instance, runes []rune) ([]rune, int, bool) {
	start, end, ok := textAreaSelectionRange(inst, len(runes))
	if !ok {
		return nil, 0, false
	}
	next := make([]rune, 0, len(runes)-(end-start))
	next = append(next, runes[:start]...)
	next = append(next, runes[end:]...)
	return next, start, true
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

	moveCursor := func(next int, extendSelection bool) bool {
		next = clampIntRuntime(next, 0, len(runes))
		if extendSelection {
			if inst.textAreaSelectionAnchor < 0 {
				inst.textAreaSelectionAnchor = inst.textAreaCursor
			}
		} else {
			textAreaClearSelection(inst)
		}
		if next == inst.textAreaCursor && !extendSelection {
			return false
		}
		inst.textAreaCursor = next
		inst.textAreaEnsureCursorVisible = true
		textAreaEnsureCursorVisible(inst, lines, runes, viewportW)
		return true
	}

	replace := func(nextRunes []rune, nextCursor int) bool {
		if opts.OnChange == nil {
			return false
		}
		opts.OnChange(string(nextRunes))
		inst.textAreaCursor = clampIntRuntime(nextCursor, 0, len(nextRunes))
		textAreaClearSelection(inst)
		inst.textAreaEnsureCursorVisible = true
		return true
	}

	insertRunes := func(inserted []rune) bool {
		base := runes
		insertAt := cursor
		if nextBase, nextCursor, deleted := textAreaDeleteSelectionRunes(inst, runes); deleted {
			base = nextBase
			insertAt = nextCursor
		}
		next := make([]rune, 0, len(base)+len(inserted))
		next = append(next, base[:insertAt]...)
		next = append(next, inserted...)
		next = append(next, base[insertAt:]...)
		return replace(next, insertAt+len(inserted))
	}

	extend := press.Mod&input.ModShift != 0

	switch press.Key {
	case input.KeyCtrlC:
		selected, ok := textAreaSelectedText(inst, runes)
		if !ok {
			return false
		}
		if opts.OnCopy != nil {
			opts.OnCopy(selected)
		}
		return true

	case input.KeyLeft:
		return moveCursor(cursor-1, extend)

	case input.KeyRight:
		return moveCursor(cursor+1, extend)

	case input.KeyUp:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		if row <= 0 {
			return false
		}
		return moveCursor(textAreaCursorIndexAt(lines, runes, row-1, x), extend)

	case input.KeyDown:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		if row >= len(lines)-1 {
			return false
		}
		return moveCursor(textAreaCursorIndexAt(lines, runes, row+1, x), extend)

	case input.KeyPgUp:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		nextRow := row - content.H
		if nextRow < 0 {
			nextRow = 0
		}
		return moveCursor(textAreaCursorIndexAt(lines, runes, nextRow, x), extend)

	case input.KeyPgDown:
		row, x := textAreaCursorLineAndX(lines, runes, cursor)
		nextRow := row + content.H
		if nextRow >= len(lines) {
			nextRow = len(lines) - 1
		}
		return moveCursor(textAreaCursorIndexAt(lines, runes, nextRow, x), extend)

	case input.KeyHome:
		row, _ := textAreaCursorLineAndX(lines, runes, cursor)
		return moveCursor(lines[row].Start, extend)

	case input.KeyEnd:
		row, _ := textAreaCursorLineAndX(lines, runes, cursor)
		return moveCursor(lines[row].End, extend)

	case input.KeyBackspace:
		if _, _, hasSelection := textAreaSelectionRange(inst, len(runes)); hasSelection {
			if nextBase, nextCursor, deleted := textAreaDeleteSelectionRunes(inst, runes); deleted {
				return replace(nextBase, nextCursor)
			}
		}
		if cursor == 0 {
			return false
		}
		next := make([]rune, 0, len(runes)-1)
		next = append(next, runes[:cursor-1]...)
		next = append(next, runes[cursor:]...)
		return replace(next, cursor-1)

	case input.KeyDelete:
		if _, _, hasSelection := textAreaSelectionRange(inst, len(runes)); hasSelection {
			if nextBase, nextCursor, deleted := textAreaDeleteSelectionRunes(inst, runes); deleted {
				return replace(nextBase, nextCursor)
			}
		}
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
		return insertRunes([]rune{'\n'})

	case input.KeyRune:
		if press.Rune == 0 {
			return false
		}
		return insertRunes([]rune{press.Rune})
	}

	return false
}

func handleTextAreaPaste(inst *Instance, text string) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.TextAreaKind {
		return false
	}
	if inst.nd.Disabled {
		return false
	}
	opts := inst.nd.TextAreaOpts
	if opts.OnChange == nil {
		return false
	}

	text = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text)
	pastedRunes := []rune(text)
	if len(pastedRunes) == 0 {
		return false
	}

	value := inst.nd.Text
	runes := []rune(value)

	base := runes
	insertAt := clampIntRuntime(inst.textAreaCursor, 0, len(runes))
	if nextBase, nextCursor, deleted := textAreaDeleteSelectionRunes(inst, runes); deleted {
		base = nextBase
		insertAt = nextCursor
	}

	next := make([]rune, 0, len(base)+len(pastedRunes))
	next = append(next, base[:insertAt]...)
	next = append(next, pastedRunes...)
	next = append(next, base[insertAt:]...)

	opts.OnChange(string(next))
	inst.textAreaCursor = insertAt + len(pastedRunes)
	textAreaClearSelection(inst)
	inst.textAreaEnsureCursorVisible = true

	return true
}

// handleTextAreaPointer converts local mouse coordinates to a rune index and
// updates cursor and selection anchor accordingly.
func handleTextAreaPointer(inst *Instance, localX, localY int, extendSelection bool) bool {
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

	idx := textAreaCursorIndexAt(lines, runes, row, x)

	if extendSelection {
		if inst.textAreaSelectionAnchor < 0 {
			inst.textAreaSelectionAnchor = inst.textAreaCursor
		}
	} else {
		inst.textAreaSelectionAnchor = idx
	}

	inst.textAreaCursor = idx
	inst.textAreaEnsureCursorVisible = true
	textAreaEnsureCursorVisible(inst, lines, runes, viewportW)
	return true
}

func handleTextAreaClick(inst *Instance, localX, localY int) bool {
	textAreaClearSelection(inst)
	inst.textAreaDragging = true
	return handleTextAreaPointer(inst, localX, localY, false)
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

	if inst.scrollY != oldY {
		inst.textAreaEnsureCursorVisible = false
		return true
	}
	return false
}

// drawTextAreaLine renders one visual line of the textarea with optional selection highlighting.
func drawTextAreaLine(
	buf *screen.Buffer,
	content style.Rect,
	viewportW int,
	line textAreaVisualLine,
	valueRunes []rune,
	scrollX int,
	wordWrap bool,
	textStyle screen.CellStyle,
	selectedStyle screen.CellStyle,
	selStart int,
	selEnd int,
	hasSelection bool,
	y int,
) {
	if !hasSelection || selEnd <= line.Start || selStart >= line.End {
		if wordWrap {
			buf.DrawTextClipped(content.X, y, line.Text, textStyle, content.X, content.Y, viewportW, content.H)
		} else {
			buf.DrawTextClipped(content.X-scrollX, y, line.Text, textStyle, content.X, content.Y, viewportW, content.H)
		}
		return
	}

	a := selStart
	if a < line.Start {
		a = line.Start
	}
	b := selEnd
	if b > line.End {
		b = line.End
	}

	pre := string(valueRunes[line.Start:a])
	mid := string(valueRunes[a:b])
	post := string(valueRunes[b:line.End])

	preWidth := runewidth.StringWidth(pre)
	midWidth := runewidth.StringWidth(mid)

	baseX := content.X
	if !wordWrap {
		baseX = content.X - scrollX
	}

	buf.DrawTextClipped(baseX, y, pre, textStyle, content.X, content.Y, viewportW, content.H)
	buf.DrawTextClipped(baseX+preWidth, y, mid, selectedStyle, content.X, content.Y, viewportW, content.H)
	buf.DrawTextClipped(baseX+preWidth+midWidth, y, post, textStyle, content.X, content.Y, viewportW, content.H)
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

	if inst.textAreaEnsureCursorVisible {
		textAreaEnsureCursorVisible(inst, cursorLines, valueRunes, viewportW)
		inst.textAreaEnsureCursorVisible = false
	}

	textStyle := screenCellStyleFromStyle(s)
	selectedStyle := textStyle
	selectedStyle.Reverse = true

	// Determine selection range; only apply to real value, not placeholder text.
	selStart, selEnd, hasSelection := 0, 0, false
	if value != "" {
		selStart, selEnd, hasSelection = textAreaSelectionRange(inst, len(valueRunes))
	}

	for row := 0; row < content.H; row++ {
		lineIdx := inst.scrollY + row
		if lineIdx >= len(lines) {
			break
		}

		line := lines[lineIdx]
		y := content.Y + row

		drawTextAreaLine(buf, content, viewportW, line, valueRunes,
			inst.scrollX, opts.WordWrap, textStyle, selectedStyle,
			selStart, selEnd, hasSelection, y)
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
