// Command scrollabletree demonstrates the ScrollableTree widget with lazy
// filesystem children.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	tui "github.com/smasonuk/earlgray"
)

var browser = newFileBrowserStore(".")

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func App() tui.Node {
	return tui.Component(AppRoot)
}

func AppRoot() tui.Node {
	selected, setSelected := tui.UseState(browser.root)
	expanded, _, updateExpanded := tui.UseStateWithUpdater(map[string]bool{
		browser.root: true,
	})
	checked, _, updateChecked := tui.UseStateWithUpdater(map[string]bool{})

	status := "Selected: " + browser.label(selected)
	if checkedCount := countChecked(checked); checkedCount > 0 {
		status = fmt.Sprintf("%s  Checked: %d", status, checkedCount)
	}

	return tui.View(
		tui.Style{
			Direction: tui.Column,
			FlexGrow:  1,
			Padding:   tui.All(1),
			Gap:       1,
		},
		tui.Text("ScrollableTree File Browser", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.Text("Up/Down navigate  Left/Right collapse/expand  Space checks  Enter activates  Ctrl-C quits"),
		tui.ScrollableTree(tui.ScrollableTreeProps{
			Roots:       browser.roots(),
			SelectedID:  selected,
			Expanded:    expanded,
			Checked:     checked,
			GetChildren: browser.children,
			OnSelect:    setSelected,
			OnExpandedChange: func(id string, open bool) {
				updateExpanded(func(m map[string]bool) map[string]bool {
					next := cloneBoolMap(m)
					next[id] = open
					return next
				})
			},
			OnCheckedChange: func(id string, value bool) {
				updateChecked(func(m map[string]bool) map[string]bool {
					next := cloneBoolMap(m)
					next[id] = value
					return next
				})
			},
			ShowFooter: true,
			AutoFocus:  true,
			Style: tui.Style{
				FlexGrow: 1,
				Border:   tui.BorderAll,
			},
			FocusedStyle: tui.Style{
				Foreground: tui.ANSIColor(3),
			},
		}),
		tui.Text(status),
	)
}

type fileBrowserStore struct {
	root  string
	cache map[string][]tui.ScrollableTreeItem
}

func newFileBrowserStore(root string) *fileBrowserStore {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &fileBrowserStore{
		root:  filepath.Clean(abs),
		cache: make(map[string][]tui.ScrollableTreeItem),
	}
}

func (s *fileBrowserStore) roots() []tui.ScrollableTreeItem {
	return []tui.ScrollableTreeItem{
		{
			ID:       s.root,
			Label:    s.label(s.root),
			IsBranch: true,
		},
	}
}

func (s *fileBrowserStore) children(id string) []tui.ScrollableTreeItem {
	if items, ok := s.cache[id]; ok {
		return items
	}

	entries, err := os.ReadDir(id)
	if err != nil {
		items := []tui.ScrollableTreeItem{
			{
				ID:       id + "#read-error",
				Label:    "(unable to read)",
				Disabled: true,
			},
		}
		s.cache[id] = items
		return items
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	items := make([]tui.ScrollableTreeItem, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(id, entry.Name())
		items = append(items, tui.ScrollableTreeItem{
			ID:       path,
			Label:    entry.Name(),
			IsBranch: entry.IsDir(),
		})
	}

	s.cache[id] = items
	return items
}

func (s *fileBrowserStore) label(id string) string {
	rel, err := filepath.Rel(s.root, id)
	if err == nil && rel != "." && rel != "" {
		return rel
	}
	base := filepath.Base(s.root)
	if base == "." || base == string(filepath.Separator) {
		return s.root
	}
	return base
}

func cloneBoolMap(m map[string]bool) map[string]bool {
	next := make(map[string]bool, len(m)+1)
	for k, v := range m {
		next[k] = v
	}
	return next
}

func countChecked(m map[string]bool) int {
	count := 0
	for _, checked := range m {
		if checked {
			count++
		}
	}
	return count
}
