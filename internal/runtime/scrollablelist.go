package runtime

import (
	"fmt"

	"github.com/smasonuk/earlgray/internal/input"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/screen"
	"github.com/smasonuk/earlgray/internal/style"
)

func resetScrollableListState(inst *Instance) {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableListKind {
		return
	}
	inst.scrollY = 0
}

func handleScrollableListKey(inst *Instance, press input.KeyPress) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableListKind || inst.nd.Disabled {
		return false
	}

	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	if count == 0 {
		return false
	}

	selected := scrollableListClampIndex(opts.SelectedIndex, count)
	visibleRows := scrollableListViewportRows(inst)

	switch press.Key {
	case input.KeyUp:
		return scrollableListSelect(inst, selected-1)
	case input.KeyDown:
		return scrollableListSelect(inst, selected+1)
	case input.KeyPgUp:
		return scrollableListSelect(inst, selected-visibleRows)
	case input.KeyPgDown:
		return scrollableListSelect(inst, selected+visibleRows)
	case input.KeyHome:
		return scrollableListSelect(inst, 0)
	case input.KeyEnd:
		return scrollableListSelect(inst, count-1)
	case input.KeyEnter:
		return scrollableListActivate(inst)
	}

	return false
}

func scrollableListSelect(inst *Instance, index int) bool {
	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	if inst.nd.Disabled || count == 0 || opts.OnSelect == nil {
		return false
	}

	current := scrollableListClampIndex(opts.SelectedIndex, count)
	next := scrollableListClampIndex(index, count)
	if next == current {
		return false
	}

	visibleRows := scrollableListViewportRows(inst)
	inst.scrollY = scrollableListEnsureVisible(inst.scrollY, next, visibleRows, count)
	opts.OnSelect(next)
	return true
}

func scrollableListActivate(inst *Instance) bool {
	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	if inst.nd.Disabled || count == 0 {
		return false
	}

	selected := scrollableListClampIndex(opts.SelectedIndex, count)
	if opts.OnActivate != nil {
		opts.OnActivate(selected)
		return true
	}
	if opts.OnSelect != nil {
		opts.OnSelect(selected)
		return true
	}
	return false
}

func handleScrollableListClick(inst *Instance, localY int) bool {
	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	if inst.nd.Disabled || count == 0 || (opts.OnClick == nil && opts.OnSelect == nil) {
		return false
	}

	visibleRows := scrollableListViewportRows(inst)
	if localY < 0 || localY >= visibleRows {
		return false
	}

	index := inst.scrollY + localY
	if index < 0 || index >= count {
		return false
	}

	if opts.OnClick != nil {
		opts.OnClick(index)
		return true
	}

	opts.OnSelect(index)
	return true
}

func renderScrollableList(inst *Instance, buf *screen.Buffer, content style.Rect, s style.Style) {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableListKind {
		return
	}
	if content.W <= 0 || content.H <= 0 {
		return
	}

	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	footerRows := scrollableListFooterRows(inst)
	visibleRows := content.H - footerRows
	if visibleRows < 1 {
		visibleRows = 1
	}

	selected := scrollableListClampIndex(opts.SelectedIndex, count)
	inst.scrollY = scrollableListEnsureVisible(inst.scrollY, selected, visibleRows, count)

	textStyle := screenCellStyleFromStyle(s)

	if count == 0 {
		empty := opts.EmptyText
		if empty == "" {
			empty = "No items."
		}
		buf.DrawTextClipped(content.X, content.Y, empty, textStyle, content.X, content.Y, content.W, content.H)
		return
	}

	end := inst.scrollY + visibleRows
	if end > count {
		end = count
	}

	rowY := content.Y
	for i := inst.scrollY; i < end; i++ {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		buf.DrawTextClipped(
			content.X,
			rowY,
			prefix+opts.Items[i].Label,
			textStyle,
			content.X,
			content.Y,
			content.W,
			content.H,
		)
		rowY++
	}

	if footerRows > 0 {
		start := inst.scrollY + 1
		buf.DrawTextClipped(
			content.X,
			content.Y+content.H-1,
			fmt.Sprintf("showing %d-%d of %d", start, end, count),
			textStyle,
			content.X,
			content.Y,
			content.W,
			content.H,
		)
	}
}

func scrollableListViewportRows(inst *Instance) int {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableListKind {
		return 1
	}

	content := inst.layout.Content
	if content.H <= 0 {
		return 1
	}

	rows := content.H - scrollableListFooterRows(inst)
	if rows < 1 {
		return 1
	}
	return rows
}

func scrollableListFooterRows(inst *Instance) int {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableListKind {
		return 0
	}

	content := inst.layout.Content
	opts := inst.nd.ScrollableListOpts
	count := len(opts.Items)
	if opts.ShowFooter && count > content.H && content.H > 1 {
		return 1
	}
	return 0
}

func scrollableListEnsureVisible(offset, selected, visibleRows, count int) int {
	if count <= 0 || visibleRows <= 0 {
		return 0
	}

	if selected < offset {
		offset = selected
	}
	if selected >= offset+visibleRows {
		offset = selected - visibleRows + 1
	}

	return clampIntRuntime(offset, 0, scrollableListMaxOffset(count, visibleRows))
}

func scrollableListClampIndex(index, count int) int {
	if count <= 0 {
		return 0
	}
	return clampIntRuntime(index, 0, count-1)
}

func scrollableListMaxOffset(count, visibleRows int) int {
	maxOffset := count - visibleRows
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

func overlayRuntimeVisualStyle(base, focus style.Style) style.Style {
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
