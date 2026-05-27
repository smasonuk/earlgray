package runtime

import (
	"fmt"
	"strings"

	"github.com/smasonuk/earlgray/internal/input"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/screen"
	"github.com/smasonuk/earlgray/internal/style"
)

type scrollableTreeRow struct {
	item   node.ScrollableTreeItem
	depth  int
	parent string
}

func resetScrollableTreeState(inst *Instance) {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind {
		return
	}
	inst.scrollY = 0
}

func scrollableTreeVisibleRows(opts node.ScrollableTreeOptions) []scrollableTreeRow {
	var rows []scrollableTreeRow

	var walk func(item node.ScrollableTreeItem, depth int, parent string, ancestors []string)
	walk = func(item node.ScrollableTreeItem, depth int, parent string, ancestors []string) {
		rows = append(rows, scrollableTreeRow{item: item, depth: depth, parent: parent})
		if !item.IsBranch || !opts.Expanded[item.ID] || opts.GetChildren == nil {
			return
		}

		nextAncestors := append(ancestors, item.ID)
		for _, child := range opts.GetChildren(item.ID) {
			if scrollableTreeContainsID(nextAncestors, child.ID) {
				continue
			}
			walk(child, depth+1, item.ID, nextAncestors)
		}
	}

	for _, root := range opts.Roots {
		walk(root, 0, "", nil)
	}
	return rows
}

func handleScrollableTreeKey(inst *Instance, press input.KeyPress) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind || inst.nd.Disabled {
		return false
	}

	opts := inst.nd.ScrollableTreeOpts
	rows := scrollableTreeVisibleRows(opts)
	if len(rows) == 0 {
		return false
	}

	selected := scrollableTreeSelectedIndex(rows, opts.SelectedID)
	visibleRows := scrollableTreeViewportRowsForCount(inst, len(rows))

	switch press.Key {
	case input.KeyUp:
		return scrollableTreeSelect(inst, rows, scrollableTreePrevEnabledIndex(rows, selected))
	case input.KeyDown:
		return scrollableTreeSelect(inst, rows, scrollableTreeNextEnabledIndex(rows, selected))
	case input.KeyPgUp:
		target := selected - visibleRows
		if target < 0 {
			target = 0
		}
		next := scrollableTreeEnabledAtOrBefore(rows, target)
		if next < 0 {
			next = scrollableTreeFirstEnabledIndex(rows)
		}
		return scrollableTreeSelect(inst, rows, next)
	case input.KeyPgDown:
		target := selected + visibleRows
		if target >= len(rows) {
			target = len(rows) - 1
		}
		next := scrollableTreeEnabledAtOrAfter(rows, target)
		if next < 0 {
			next = scrollableTreeLastEnabledIndex(rows)
		}
		return scrollableTreeSelect(inst, rows, next)
	case input.KeyHome:
		return scrollableTreeSelect(inst, rows, scrollableTreeFirstEnabledIndex(rows))
	case input.KeyEnd:
		return scrollableTreeSelect(inst, rows, scrollableTreeLastEnabledIndex(rows))
	case input.KeyRight:
		return scrollableTreeRight(inst, rows, selected)
	case input.KeyLeft:
		return scrollableTreeLeft(inst, rows, selected)
	case input.KeyRune:
		if press.Rune == ' ' {
			return scrollableTreeToggleChecked(inst, rows, selected)
		}
	case input.KeyEnter:
		return scrollableTreeActivate(inst, rows, selected)
	}

	return false
}

func scrollableTreeSelect(inst *Instance, rows []scrollableTreeRow, index int) bool {
	if inst == nil || inst.nd == nil || inst.nd.Disabled || index < 0 || index >= len(rows) {
		return false
	}

	row := rows[index]
	if row.item.Disabled || inst.nd.ScrollableTreeOpts.OnSelect == nil {
		return false
	}
	if inst.nd.ScrollableTreeOpts.SelectedID == row.item.ID {
		return false
	}

	visibleRows := scrollableTreeViewportRowsForCount(inst, len(rows))
	inst.scrollY = scrollableTreeEnsureVisible(inst.scrollY, index, visibleRows, len(rows))
	inst.nd.ScrollableTreeOpts.OnSelect(row.item.ID)
	return true
}

func scrollableTreeRight(inst *Instance, rows []scrollableTreeRow, selected int) bool {
	if selected < 0 || selected >= len(rows) {
		return false
	}
	opts := inst.nd.ScrollableTreeOpts
	row := rows[selected]
	if row.item.Disabled || !row.item.IsBranch {
		return false
	}

	if !opts.Expanded[row.item.ID] {
		if opts.OnExpandedChange == nil {
			return false
		}
		opts.OnExpandedChange(row.item.ID, true)
		return true
	}

	child := scrollableTreeFirstEnabledChildIndex(rows, selected)
	if child < 0 {
		return false
	}
	return scrollableTreeSelect(inst, rows, child)
}

func scrollableTreeLeft(inst *Instance, rows []scrollableTreeRow, selected int) bool {
	if selected < 0 || selected >= len(rows) {
		return false
	}
	opts := inst.nd.ScrollableTreeOpts
	row := rows[selected]
	if row.item.Disabled {
		return false
	}

	if row.item.IsBranch && opts.Expanded[row.item.ID] {
		if opts.OnExpandedChange == nil {
			return false
		}
		opts.OnExpandedChange(row.item.ID, false)
		return true
	}

	parent := scrollableTreeParentIndex(rows, selected)
	if parent < 0 {
		return false
	}
	return scrollableTreeSelect(inst, rows, parent)
}

func scrollableTreeToggleChecked(inst *Instance, rows []scrollableTreeRow, selected int) bool {
	if selected < 0 || selected >= len(rows) {
		return false
	}
	opts := inst.nd.ScrollableTreeOpts
	row := rows[selected]
	if row.item.Disabled || opts.OnCheckedChange == nil {
		return false
	}
	opts.OnCheckedChange(row.item.ID, !opts.Checked[row.item.ID])
	return true
}

func scrollableTreeActivate(inst *Instance, rows []scrollableTreeRow, selected int) bool {
	if selected < 0 || selected >= len(rows) {
		return false
	}
	opts := inst.nd.ScrollableTreeOpts
	row := rows[selected]
	if row.item.Disabled {
		return false
	}

	if opts.OnActivate != nil {
		opts.OnActivate(row.item.ID)
		return true
	}
	if row.item.IsBranch {
		if opts.OnExpandedChange == nil {
			return false
		}
		opts.OnExpandedChange(row.item.ID, !opts.Expanded[row.item.ID])
		return true
	}
	if opts.OnSelect != nil {
		opts.OnSelect(row.item.ID)
		return true
	}
	return false
}

func handleScrollableTreeClick(inst *Instance, localX, localY int) bool {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind || inst.nd.Disabled {
		return false
	}
	if localX < 0 || localY < 0 {
		return false
	}

	opts := inst.nd.ScrollableTreeOpts
	rows := scrollableTreeVisibleRows(opts)
	if len(rows) == 0 {
		return false
	}

	visibleRows := scrollableTreeViewportRowsForCount(inst, len(rows))
	if localY >= visibleRows {
		return false
	}

	index := inst.scrollY + localY
	if index < 0 || index >= len(rows) {
		return false
	}

	row := rows[index]
	if row.item.Disabled {
		return false
	}

	disclosureStart := 2 + row.depth*2
	checkboxStart := disclosureStart + 2
	labelStart := checkboxStart + 4

	if localX >= disclosureStart && localX < checkboxStart {
		if !row.item.IsBranch || opts.OnExpandedChange == nil {
			return false
		}
		opts.OnExpandedChange(row.item.ID, !opts.Expanded[row.item.ID])
		return true
	}

	if localX >= checkboxStart && localX < labelStart {
		if opts.OnCheckedChange == nil {
			return false
		}
		opts.OnCheckedChange(row.item.ID, !opts.Checked[row.item.ID])
		return true
	}

	if localX >= labelStart {
		if opts.OnSelect == nil {
			return false
		}
		opts.OnSelect(row.item.ID)
		return true
	}

	return false
}

func renderScrollableTree(inst *Instance, buf *screen.Buffer, content style.Rect, s style.Style) {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind {
		return
	}
	if content.W <= 0 || content.H <= 0 {
		return
	}

	opts := inst.nd.ScrollableTreeOpts
	rows := scrollableTreeVisibleRows(opts)
	count := len(rows)
	footerRows := scrollableTreeFooterRows(inst, count)
	visibleRows := content.H - footerRows
	if visibleRows < 1 {
		visibleRows = 1
	}

	selected := scrollableTreeSelectedIndex(rows, opts.SelectedID)
	inst.scrollY = scrollableTreeEnsureVisible(inst.scrollY, selected, visibleRows, count)

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
		buf.DrawTextClipped(
			content.X,
			rowY,
			scrollableTreeRowText(rows[i], opts, i == selected),
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

func scrollableTreeRowText(row scrollableTreeRow, opts node.ScrollableTreeOptions, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	disclosure := "  "
	if row.item.IsBranch {
		disclosure = "▸ "
		if opts.Expanded[row.item.ID] {
			disclosure = "▾ "
		}
	}

	checkbox := "[ ] "
	if opts.Checked[row.item.ID] {
		checkbox = "[x] "
	}

	return prefix + strings.Repeat(" ", row.depth*2) + disclosure + checkbox + row.item.Label
}

func scrollableTreeViewportRows(inst *Instance) int {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind {
		return 1
	}
	count := len(scrollableTreeVisibleRows(inst.nd.ScrollableTreeOpts))
	return scrollableTreeViewportRowsForCount(inst, count)
}

func scrollableTreeViewportRowsForCount(inst *Instance, count int) int {
	content := inst.layout.Content
	if content.H <= 0 {
		return 1
	}

	rows := content.H - scrollableTreeFooterRows(inst, count)
	if rows < 1 {
		return 1
	}
	return rows
}

func scrollableTreeFooterRows(inst *Instance, count int) int {
	if inst == nil || inst.nd == nil || inst.nd.Kind != node.ScrollableTreeKind {
		return 0
	}

	content := inst.layout.Content
	opts := inst.nd.ScrollableTreeOpts
	if opts.ShowFooter && count > content.H && content.H > 1 {
		return 1
	}
	return 0
}

func scrollableTreeEnsureVisible(offset, selected, visibleRows, count int) int {
	if count <= 0 || visibleRows <= 0 {
		return 0
	}

	selected = scrollableTreeClampIndex(selected, count)
	if selected < offset {
		offset = selected
	}
	if selected >= offset+visibleRows {
		offset = selected - visibleRows + 1
	}

	return clampIntRuntime(offset, 0, scrollableTreeMaxOffset(count, visibleRows))
}

func scrollableTreeSelectedIndex(rows []scrollableTreeRow, selectedID string) int {
	for i, row := range rows {
		if row.item.ID == selectedID {
			return i
		}
	}
	return scrollableTreeClampIndex(0, len(rows))
}

func scrollableTreeClampIndex(index, count int) int {
	if count <= 0 {
		return 0
	}
	return clampIntRuntime(index, 0, count-1)
}

func scrollableTreeMaxOffset(count, visibleRows int) int {
	maxOffset := count - visibleRows
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

func scrollableTreeFirstEnabledIndex(rows []scrollableTreeRow) int {
	return scrollableTreeEnabledAtOrAfter(rows, 0)
}

func scrollableTreeLastEnabledIndex(rows []scrollableTreeRow) int {
	return scrollableTreeEnabledAtOrBefore(rows, len(rows)-1)
}

func scrollableTreePrevEnabledIndex(rows []scrollableTreeRow, from int) int {
	return scrollableTreeEnabledAtOrBefore(rows, from-1)
}

func scrollableTreeNextEnabledIndex(rows []scrollableTreeRow, from int) int {
	return scrollableTreeEnabledAtOrAfter(rows, from+1)
}

func scrollableTreeEnabledAtOrBefore(rows []scrollableTreeRow, from int) int {
	if from >= len(rows) {
		from = len(rows) - 1
	}
	for i := from; i >= 0; i-- {
		if !rows[i].item.Disabled {
			return i
		}
	}
	return -1
}

func scrollableTreeEnabledAtOrAfter(rows []scrollableTreeRow, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < len(rows); i++ {
		if !rows[i].item.Disabled {
			return i
		}
	}
	return -1
}

func scrollableTreeFirstEnabledChildIndex(rows []scrollableTreeRow, parent int) int {
	if parent < 0 || parent >= len(rows) {
		return -1
	}
	parentDepth := rows[parent].depth
	for i := parent + 1; i < len(rows); i++ {
		if rows[i].depth <= parentDepth {
			return -1
		}
		if rows[i].depth == parentDepth+1 && rows[i].parent == rows[parent].item.ID && !rows[i].item.Disabled {
			return i
		}
	}
	return -1
}

func scrollableTreeParentIndex(rows []scrollableTreeRow, child int) int {
	if child < 0 || child >= len(rows) {
		return -1
	}
	parentID := rows[child].parent
	if parentID == "" {
		return -1
	}
	for i := child - 1; i >= 0; i-- {
		if rows[i].item.ID == parentID {
			if rows[i].item.Disabled {
				return -1
			}
			return i
		}
	}
	return -1
}

func scrollableTreeContainsID(values []string, id string) bool {
	for _, value := range values {
		if value == id {
			return true
		}
	}
	return false
}
