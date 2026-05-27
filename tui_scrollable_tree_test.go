package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smasonuk/earlgray/internal/event"
	inode "github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/runtime"
	"github.com/smasonuk/earlgray/internal/screen"
)

func TestScrollableTreeReturnsNativeNode(t *testing.T) {
	got := ScrollableTree(ScrollableTreeProps{})
	if got.Kind != inode.ScrollableTreeKind {
		t.Fatalf("ScrollableTree should return ScrollableTreeKind, got %v", got.Kind)
	}
	if !got.Focusable {
		t.Fatal("ScrollableTree should be focusable when enabled")
	}

	disabled := ScrollableTree(ScrollableTreeProps{Disabled: true})
	if disabled.Focusable {
		t.Fatal("disabled ScrollableTree should not be focusable")
	}
}

func TestScrollableTreeCollapsedByDefault(t *testing.T) {
	calls := 0
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots: scrollableTreeTestRoots(),
		GetChildren: func(id string) []ScrollableTreeItem {
			calls++
			return testTreeChildren(id)
		},
	}))

	assertScrollableTreeContains(t, text, "▸ [ ] root")
	assertScrollableTreeNotContains(t, text, "README.md")
	if calls != 0 {
		t.Fatalf("GetChildren called %d times for collapsed root, want 0", calls)
	}
}

func TestScrollableTreeRendersExpandedChildren(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		Expanded:    map[string]bool{"root": true},
		GetChildren: testTreeChildren,
	}))

	assertScrollableTreeContains(t, text, "▾ [ ] root")
	assertScrollableTreeContains(t, text, "    ▸ [ ] src")
	assertScrollableTreeContains(t, text, "      [ ] README.md")
}

func TestScrollableTreeRendersCheckedNodes(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		Expanded:    map[string]bool{"root": true},
		Checked:     map[string]bool{"readme": true},
		GetChildren: testTreeChildren,
	}))

	assertScrollableTreeContains(t, text, "[x] README.md")
	assertScrollableTreeContains(t, text, "[ ] root")
}

func TestScrollableTreeCallsGetChildrenOnlyForExpandedBranches(t *testing.T) {
	called := map[string]int{}
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{
			{ID: "root", Label: "root", IsBranch: true},
			{ID: "other", Label: "other", IsBranch: true},
		},
		Expanded: map[string]bool{"root": true},
		GetChildren: func(id string) []ScrollableTreeItem {
			called[id]++
			return []ScrollableTreeItem{{ID: id + "-child", Label: id + "-child"}}
		},
	}))

	assertScrollableTreeContains(t, text, "root-child")
	assertScrollableTreeNotContains(t, text, "other-child")
	if len(called) != 1 || called["root"] == 0 {
		t.Fatalf("GetChildren calls = %#v, want only root", called)
	}
}

func TestScrollableTreeDoesNotWalkCollapsedLargeTree(t *testing.T) {
	calls := 0
	_ = renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{{ID: "root", Label: "root", IsBranch: true}},
		GetChildren: func(string) []ScrollableTreeItem {
			calls++
			items := make([]ScrollableTreeItem, 10000)
			for i := range items {
				items[i] = ScrollableTreeItem{ID: fmt.Sprintf("item-%d", i), Label: fmt.Sprintf("item %d", i)}
			}
			return items
		},
	}))

	if calls != 0 {
		t.Fatalf("GetChildren called %d times for collapsed large tree, want 0", calls)
	}
}

func TestScrollableTreeDownSelectsNextVisibleRow(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:      scrollableTreeFlatRoots(3),
		SelectedID: "item-0",
		AutoFocus:  true,
		OnSelect:   func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}})
	if !consumed {
		t.Fatal("Down should be consumed")
	}
	if got != "item-1" {
		t.Fatalf("OnSelect = %q, want item-1", got)
	}
}

func TestScrollableTreeUpAtTopReturnsFalse(t *testing.T) {
	called := false
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:      scrollableTreeFlatRoots(3),
		SelectedID: "item-0",
		AutoFocus:  true,
		OnSelect:   func(string) { called = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyUp}})
	if consumed {
		t.Fatal("Up at first item should return false")
	}
	if called {
		t.Fatal("OnSelect should not be called")
	}
}

func TestScrollableTreePageDownMovesByVisibleRows(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(10),
		SelectedID:  "item-0",
		VisibleRows: 4,
		AutoFocus:   true,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyPgDn}})
	if !consumed {
		t.Fatal("PgDown should be consumed")
	}
	if got != "item-4" {
		t.Fatalf("OnSelect = %q, want item-4", got)
	}
}

func TestScrollableTreeHomeEnd(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(10),
		SelectedID:  "item-5",
		VisibleRows: 4,
		AutoFocus:   true,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}}) {
		t.Fatal("Home should be consumed")
	}
	if got != "item-0" {
		t.Fatalf("Home OnSelect = %q, want item-0", got)
	}

	got = ""
	rt = runtime.New()
	root = ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(10),
		SelectedID:  "item-5",
		VisibleRows: 4,
		AutoFocus:   true,
		OnSelect:    func(id string) { got = id },
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}}) {
		t.Fatal("End should be consumed")
	}
	if got != "item-9" {
		t.Fatalf("End OnSelect = %q, want item-9", got)
	}
}

func TestScrollableTreeNavigationSkipsCollapsedChildren(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{
			{ID: "root", Label: "root", IsBranch: true},
			{ID: "sibling", Label: "sibling"},
		},
		SelectedID:  "root",
		AutoFocus:   true,
		GetChildren: testTreeChildren,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}}) {
		t.Fatal("Down should be consumed")
	}
	if got != "sibling" {
		t.Fatalf("OnSelect = %q, want sibling", got)
	}
}

func TestScrollableTreeNavigationIncludesExpandedChildren(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		SelectedID:  "root",
		Expanded:    map[string]bool{"root": true},
		AutoFocus:   true,
		GetChildren: testTreeChildren,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}}) {
		t.Fatal("Down should be consumed")
	}
	if got != "src" {
		t.Fatalf("OnSelect = %q, want src", got)
	}
}

func TestScrollableTreeRightExpandsCollapsedBranch(t *testing.T) {
	var gotID string
	var gotOpen bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		AutoFocus:        true,
		OnExpandedChange: func(id string, open bool) { gotID, gotOpen = id, open },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}}) {
		t.Fatal("Right should be consumed")
	}
	if gotID != "root" || !gotOpen {
		t.Fatalf("OnExpandedChange = (%q, %v), want (root, true)", gotID, gotOpen)
	}
}

func TestScrollableTreeRightMovesToFirstChildWhenAlreadyExpanded(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		SelectedID:  "root",
		Expanded:    map[string]bool{"root": true},
		AutoFocus:   true,
		GetChildren: testTreeChildren,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}}) {
		t.Fatal("Right should select first child")
	}
	if got != "src" {
		t.Fatalf("OnSelect = %q, want src", got)
	}
}

func TestScrollableTreeLeftCollapsesExpandedBranch(t *testing.T) {
	var gotID string
	var gotOpen bool = true
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		Expanded:         map[string]bool{"root": true},
		AutoFocus:        true,
		GetChildren:      testTreeChildren,
		OnExpandedChange: func(id string, open bool) { gotID, gotOpen = id, open },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}}) {
		t.Fatal("Left should be consumed")
	}
	if gotID != "root" || gotOpen {
		t.Fatalf("OnExpandedChange = (%q, %v), want (root, false)", gotID, gotOpen)
	}
}

func TestScrollableTreeLeftSelectsParentForLeaf(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		SelectedID:  "readme",
		Expanded:    map[string]bool{"root": true},
		AutoFocus:   true,
		GetChildren: testTreeChildren,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}}) {
		t.Fatal("Left should select parent")
	}
	if got != "root" {
		t.Fatalf("OnSelect = %q, want root", got)
	}
}

func TestScrollableTreeSpaceTogglesCheckedNode(t *testing.T) {
	var gotID string
	var gotChecked bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:           scrollableTreeFlatRoots(2),
		SelectedID:      "item-1",
		AutoFocus:       true,
		OnCheckedChange: func(id string, checked bool) { gotID, gotChecked = id, checked },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: ' '}}) {
		t.Fatal("Space should be consumed")
	}
	if gotID != "item-1" || !gotChecked {
		t.Fatalf("OnCheckedChange = (%q, %v), want (item-1, true)", gotID, gotChecked)
	}
}

func TestScrollableTreeSpaceTogglesCheckedFolder(t *testing.T) {
	var gotID string
	var gotChecked bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:           scrollableTreeTestRoots(),
		SelectedID:      "root",
		Checked:         map[string]bool{"root": true},
		AutoFocus:       true,
		OnCheckedChange: func(id string, checked bool) { gotID, gotChecked = id, checked },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: ' '}}) {
		t.Fatal("Space should be consumed")
	}
	if gotID != "root" || gotChecked {
		t.Fatalf("OnCheckedChange = (%q, %v), want (root, false)", gotID, gotChecked)
	}
}

func TestScrollableTreeDoesNotMutateControlledMaps(t *testing.T) {
	expanded := map[string]bool{"root": false}
	checked := map[string]bool{"root": false}
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		Expanded:         expanded,
		Checked:          checked,
		AutoFocus:        true,
		OnExpandedChange: func(string, bool) {},
		OnCheckedChange:  func(string, bool) {},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}}) {
		t.Fatal("Right should request expansion")
	}
	if expanded["root"] {
		t.Fatal("Expanded map was mutated by tree")
	}

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: ' '}}) {
		t.Fatal("Space should request checked change")
	}
	if checked["root"] {
		t.Fatal("Checked map was mutated by tree")
	}
}

func TestScrollableTreeEnterActivatesWhenCallbackProvided(t *testing.T) {
	activated := ""
	expanded := false
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		AutoFocus:        true,
		OnActivate:       func(id string) { activated = id },
		OnExpandedChange: func(string, bool) { expanded = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}}) {
		t.Fatal("Enter should be consumed")
	}
	if activated != "root" {
		t.Fatalf("OnActivate = %q, want root", activated)
	}
	if expanded {
		t.Fatal("OnExpandedChange should not be called when OnActivate exists")
	}
}

func TestScrollableTreeEnterTogglesBranchWhenNoActivate(t *testing.T) {
	var gotID string
	var gotOpen bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		AutoFocus:        true,
		OnExpandedChange: func(id string, open bool) { gotID, gotOpen = id, open },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}}) {
		t.Fatal("Enter should be consumed")
	}
	if gotID != "root" || !gotOpen {
		t.Fatalf("OnExpandedChange = (%q, %v), want (root, true)", gotID, gotOpen)
	}
}

func TestScrollableTreeEnterFallsBackToSelectForLeaf(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:      scrollableTreeFlatRoots(2),
		SelectedID: "item-1",
		AutoFocus:  true,
		OnSelect:   func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}}) {
		t.Fatal("Enter should be consumed")
	}
	if got != "item-1" {
		t.Fatalf("OnSelect = %q, want item-1", got)
	}
}

func TestScrollableTreeMouseClickSelectsVisibleRow(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeTestRoots(),
		Expanded:    map[string]bool{"root": true},
		GetChildren: testTreeChildren,
		OnSelect:    func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	if !rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 10, Y: 1, Button: MouseLeft}}) {
		t.Fatal("label click should be consumed")
	}
	if got != "src" {
		t.Fatalf("OnSelect = %q, want src", got)
	}
}

func TestScrollableTreeMouseClickCheckboxToggles(t *testing.T) {
	var gotID string
	var gotChecked bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:           scrollableTreeTestRoots(),
		OnCheckedChange: func(id string, checked bool) { gotID, gotChecked = id, checked },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	if !rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 4, Y: 0, Button: MouseLeft}}) {
		t.Fatal("checkbox click should be consumed")
	}
	if gotID != "root" || !gotChecked {
		t.Fatalf("OnCheckedChange = (%q, %v), want (root, true)", gotID, gotChecked)
	}
}

func TestScrollableTreeMouseClickDisclosureExpands(t *testing.T) {
	var gotID string
	var gotOpen bool
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		OnExpandedChange: func(id string, open bool) { gotID, gotOpen = id, open },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	if !rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 2, Y: 0, Button: MouseLeft}}) {
		t.Fatal("disclosure click should be consumed")
	}
	if gotID != "root" || !gotOpen {
		t.Fatalf("OnExpandedChange = (%q, %v), want (root, true)", gotID, gotOpen)
	}
}

func TestScrollableTreeMouseWheelDownSelectsNextVisibleRow(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:      scrollableTreeFlatRoots(4),
		SelectedID: "item-1",
		AutoFocus:  true,
		OnSelect:   func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	if !rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 0, Y: 0, Button: MouseWheelDown}}) {
		t.Fatal("wheel down should be consumed")
	}
	if got != "item-2" {
		t.Fatalf("OnSelect = %q, want item-2", got)
	}
}

func TestDisabledScrollableTreeIgnoresKeyboardAndMouse(t *testing.T) {
	called := false
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots:            scrollableTreeTestRoots(),
		SelectedID:       "root",
		Disabled:         true,
		OnSelect:         func(string) { called = true },
		OnCheckedChange:  func(string, bool) { called = true },
		OnExpandedChange: func(string, bool) { called = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	keyConsumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}})
	mouseConsumed := rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 8, Y: 0, Button: MouseLeft}})

	if keyConsumed || mouseConsumed || called {
		t.Fatal("disabled ScrollableTree should not consume input or call callbacks")
	}
}

func TestScrollableTreeDisabledRowsAreSkippedAndIgnoreActions(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{
			{ID: "item-0", Label: "item 0"},
			{ID: "item-1", Label: "item 1", Disabled: true},
			{ID: "item-2", Label: "item 2"},
		},
		SelectedID: "item-0",
		AutoFocus:  true,
		OnSelect:   func(id string) { got = id },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	if !rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}}) {
		t.Fatal("Down should skip disabled row and be consumed")
	}
	if got != "item-2" {
		t.Fatalf("OnSelect = %q, want item-2", got)
	}

	called := false
	rt = runtime.New()
	root = ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{
			{ID: "item-0", Label: "item 0"},
			{ID: "item-1", Label: "item 1", Disabled: true},
		},
		SelectedID:       "item-1",
		AutoFocus:        true,
		OnSelect:         func(string) { called = true },
		OnCheckedChange:  func(string, bool) { called = true },
		OnExpandedChange: func(string, bool) { called = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	keyConsumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: ' '}})
	mouseConsumed := rt.HandleEvent(event.Event{Kind: event.MouseKind, Mouse: event.Mouse{X: 8, Y: 1, Button: MouseLeft}})
	if keyConsumed || mouseConsumed || called {
		t.Fatal("disabled rows should ignore keyboard and mouse actions")
	}
}

func TestScrollableTreeRendersOnlyVisibleRows(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(10),
		VisibleRows: 3,
	}))

	assertScrollableTreeContains(t, text, "item 0")
	assertScrollableTreeContains(t, text, "item 1")
	assertScrollableTreeContains(t, text, "item 2")
	assertScrollableTreeNotContains(t, text, "item 3")
}

func TestScrollableTreeKeepsSelectedRowVisible(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(10),
		SelectedID:  "item-5",
		VisibleRows: 3,
	}))

	assertScrollableTreeContains(t, text, ">   [ ] item 5")
	assertScrollableTreeNotContains(t, text, "item 0")
	assertScrollableTreeNotContains(t, text, "item 6")
}

func TestScrollableTreeFlexGrowUsesAllocatedHeight(t *testing.T) {
	text := renderScrollableTreeTextAt(View(Style{Direction: Column},
		Text("header", WithTextStyle(Style{Height: Cells(1)})),
		ScrollableTree(ScrollableTreeProps{
			Roots: scrollableTreeFlatRoots(10),
			Style: Style{FlexGrow: 1},
		}),
		Text("footer", WithTextStyle(Style{Height: Cells(1)})),
	), 80, 6)

	assertScrollableTreeContains(t, text, "item 0")
	assertScrollableTreeContains(t, text, "item 1")
	assertScrollableTreeContains(t, text, "item 2")
	assertScrollableTreeContains(t, text, "item 3")
	assertScrollableTreeNotContains(t, text, "item 4")
}

func TestScrollableTreeDoesNotOverflowConfiguredHeight(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(12),
		VisibleRows: 8,
		Style: Style{
			Height: Cells(3),
		},
	}))

	assertScrollableTreeContains(t, text, "item 0")
	assertScrollableTreeContains(t, text, "item 1")
	assertScrollableTreeContains(t, text, "item 2")
	assertScrollableTreeNotContains(t, text, "item 3")
	assertScrollableTreeNotContains(t, text, "item 11")
}

func TestScrollableTreeFooterReservesRow(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots:       scrollableTreeFlatRoots(6),
		ShowFooter:  true,
		VisibleRows: 3,
	}))

	assertScrollableTreeContains(t, text, "item 0")
	assertScrollableTreeContains(t, text, "item 1")
	assertScrollableTreeContains(t, text, "item 2")
	assertScrollableTreeNotContains(t, text, "item 3")
	assertScrollableTreeContains(t, text, "showing 1-3 of 6")
}

func TestScrollableTreeEmptyText(t *testing.T) {
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{}))
	assertScrollableTreeContains(t, text, "No items.")

	text = renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{EmptyText: "Nothing here."}))
	assertScrollableTreeContains(t, text, "Nothing here.")
}

func TestScrollableTreeCyclicProviderDoesNotHang(t *testing.T) {
	calls := 0
	text := renderScrollableTreeText(ScrollableTree(ScrollableTreeProps{
		Roots: []ScrollableTreeItem{{ID: "root", Label: "root", IsBranch: true}},
		Expanded: map[string]bool{
			"root": true,
			"src":  true,
		},
		GetChildren: func(id string) []ScrollableTreeItem {
			calls++
			switch id {
			case "root":
				return []ScrollableTreeItem{{ID: "src", Label: "src", IsBranch: true}}
			case "src":
				return []ScrollableTreeItem{
					{ID: "root", Label: "root again", IsBranch: true},
					{ID: "main", Label: "main.go"},
				}
			default:
				return nil
			}
		},
	}))

	assertScrollableTreeContains(t, text, "src")
	assertScrollableTreeContains(t, text, "main.go")
	assertScrollableTreeNotContains(t, text, "root again")
	if calls == 0 || calls > 8 {
		t.Fatalf("GetChildren calls = %d, want finite visible traversal", calls)
	}
}

func renderScrollableTreeText(root Node) string {
	return renderScrollableTreeTextAt(root, 80, 24)
}

func renderScrollableTreeTextAt(root Node, w, h int) string {
	rt := runtime.New()
	updateUntilClean(rt, root)
	rt.RunLayout(w, h)
	buf := screen.NewBuffer(w, h)
	rt.Render(buf)
	return bufferText(buf)
}

func scrollableTreeTestRoots() []ScrollableTreeItem {
	return []ScrollableTreeItem{{ID: "root", Label: "root", IsBranch: true}}
}

func scrollableTreeFlatRoots(count int) []ScrollableTreeItem {
	items := make([]ScrollableTreeItem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, ScrollableTreeItem{
			ID:    fmt.Sprintf("item-%d", i),
			Label: fmt.Sprintf("item %d", i),
		})
	}
	return items
}

func testTreeChildren(id string) []ScrollableTreeItem {
	switch id {
	case "root":
		return []ScrollableTreeItem{
			{ID: "src", Label: "src", IsBranch: true},
			{ID: "readme", Label: "README.md"},
		}
	case "src":
		return []ScrollableTreeItem{
			{ID: "main", Label: "main.go"},
		}
	default:
		return nil
	}
}

func assertScrollableTreeContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("rendered text does not contain %q in %q", want, text)
	}
}

func assertScrollableTreeNotContains(t *testing.T, text, want string) {
	t.Helper()
	if strings.Contains(text, want) {
		t.Fatalf("rendered text contains %q in %q", want, text)
	}
}
