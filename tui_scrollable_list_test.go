package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smasonuk/earlgray/internal/event"
	"github.com/smasonuk/earlgray/internal/host"
	inode "github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/runtime"
	"github.com/smasonuk/earlgray/internal/screen"
)

func TestScrollableListReturnsNativeNode(t *testing.T) {
	got := ScrollableList(ScrollableListProps{})
	if got.Kind != inode.ScrollableListKind {
		t.Fatalf("ScrollableList should return ScrollableListKind, got %v", got.Kind)
	}
	if !got.Focusable {
		t.Fatal("ScrollableList should be focusable when enabled")
	}

	disabled := ScrollableList(ScrollableListProps{Disabled: true})
	if disabled.Focusable {
		t.Fatal("disabled ScrollableList should not be focusable")
	}
}

func TestScrollableListRendersOnlyVisibleRows(t *testing.T) {
	text := renderScrollableListText(ScrollableList(ScrollableListProps{
		Items:       scrollableListTestItems(10),
		VisibleRows: 3,
	}))

	assertScrollableListContains(t, text, "item 0")
	assertScrollableListContains(t, text, "item 1")
	assertScrollableListContains(t, text, "item 2")
	assertScrollableListNotContains(t, text, "item 3")
}

func TestScrollableListKeepsSelectedRowVisible(t *testing.T) {
	text := renderScrollableListText(ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(10),
		SelectedIndex: 5,
		VisibleRows:   3,
	}))

	assertScrollableListContains(t, text, "> item 5")
	assertScrollableListNotContains(t, text, "item 0")
	assertScrollableListNotContains(t, text, "item 6")
}

func TestScrollableListDownCallsOnSelect(t *testing.T) {
	got := -1
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(3),
		SelectedIndex: 0,
		VisibleRows:   3,
		AutoFocus:     true,
		OnSelect:      func(i int) { got = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})
	if !consumed {
		t.Fatal("Down should be consumed")
	}
	if got != 1 {
		t.Fatalf("OnSelect = %d, want 1", got)
	}
}

func TestScrollableListUpAtTopReturnsFalse(t *testing.T) {
	called := false
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(3),
		SelectedIndex: 0,
		VisibleRows:   3,
		AutoFocus:     true,
		OnSelect:      func(int) { called = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	})
	if consumed {
		t.Fatal("Up at first item should return false")
	}
	if called {
		t.Fatal("OnSelect should not be called")
	}
}

func TestScrollableListPageDownMovesByVisibleRows(t *testing.T) {
	got := -1
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(10),
		SelectedIndex: 0,
		VisibleRows:   4,
		AutoFocus:     true,
		OnSelect:      func(i int) { got = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyPgDn},
	})
	if !consumed {
		t.Fatal("PgDown should be consumed")
	}
	if got != 4 {
		t.Fatalf("OnSelect = %d, want 4", got)
	}
}

func TestScrollableListEndSelectsLastItem(t *testing.T) {
	got := -1
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(10),
		SelectedIndex: 0,
		VisibleRows:   4,
		AutoFocus:     true,
		OnSelect:      func(i int) { got = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnd},
	})
	if !consumed {
		t.Fatal("End should be consumed")
	}
	if got != 9 {
		t.Fatalf("OnSelect = %d, want 9", got)
	}
}

func TestScrollableListMouseWheelDownSelectsNextItem(t *testing.T) {
	got := -1
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(4),
		SelectedIndex: 1,
		VisibleRows:   3,
		AutoFocus:     true,
		OnSelect:      func(i int) { got = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{
		Kind: event.MouseKind,
		Mouse: event.Mouse{
			X:      0,
			Y:      0,
			Button: MouseWheelDown,
		},
	})
	if !consumed {
		t.Fatal("wheel down should be consumed")
	}
	if got != 2 {
		t.Fatalf("OnSelect = %d, want 2", got)
	}
}

func TestScrollableListPageDownUsesAllocatedHeight(t *testing.T) {
	got := -1
	rt := runtime.New()
	root := View(Style{Direction: Column},
		Text("header", WithTextStyle(Style{Height: Cells(1)})),
		ScrollableList(ScrollableListProps{
			Items:         scrollableListTestItems(10),
			SelectedIndex: 0,
			AutoFocus:     true,
			Style:         Style{FlexGrow: 1},
			OnSelect:      func(i int) { got = i },
		}),
		Text("footer", WithTextStyle(Style{Height: Cells(1)})),
	)

	updateUntilClean(rt, root)
	rt.RunLayout(80, 6)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyPgDn},
	})
	if !consumed {
		t.Fatal("PgDown should be consumed")
	}
	if got != 4 {
		t.Fatalf("OnSelect = %d, want 4 from allocated viewport height", got)
	}
}

func TestScrollableListFlexGrowUsesAllocatedHeight(t *testing.T) {
	text := renderScrollableListTextAt(View(Style{Direction: Column},
		Text("header", WithTextStyle(Style{Height: Cells(1)})),
		ScrollableList(ScrollableListProps{
			Items: scrollableListTestItems(10),
			Style: Style{FlexGrow: 1},
		}),
		Text("footer", WithTextStyle(Style{Height: Cells(1)})),
	), 80, 6)

	assertScrollableListContains(t, text, "item 0")
	assertScrollableListContains(t, text, "item 1")
	assertScrollableListContains(t, text, "item 2")
	assertScrollableListContains(t, text, "item 3")
	assertScrollableListNotContains(t, text, "item 4")
}

func TestScrollableListMouseClickSelectsVisibleRow(t *testing.T) {
	selected := -1
	app := func() Node {
		return ScrollableList(ScrollableListProps{
			Items:         scrollableListTestItems(10),
			SelectedIndex: 5,
			VisibleRows:   3,
			OnSelect:      func(i int) { selected = i },
		})
	}

	events := []event.Event{
		{
			Kind: event.MouseKind,
			Mouse: event.Mouse{
				X:      0,
				Y:      1,
				Button: MouseLeft,
			},
		},
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if selected != 4 {
		t.Fatalf("selected = %d, want 4", selected)
	}
}

func TestDisabledScrollableListIgnoresKeyboardAndMouse(t *testing.T) {
	called := false
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(3),
		SelectedIndex: 0,
		Disabled:      true,
		OnSelect:      func(int) { called = true },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	keyConsumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})
	rt.RunLayout(80, 24)
	mouseConsumed := rt.HandleEvent(event.Event{
		Kind: event.MouseKind,
		Mouse: event.Mouse{
			X:      0,
			Y:      0,
			Button: MouseLeft,
		},
	})

	if keyConsumed || mouseConsumed || called {
		t.Fatal("disabled ScrollableList should not consume input or call OnSelect")
	}
}

func TestScrollableListDoesNotOverflowConfiguredHeight(t *testing.T) {
	text := renderScrollableListText(ScrollableList(ScrollableListProps{
		Items:       scrollableListTestItems(12),
		VisibleRows: 8,
		Style: Style{
			Height: Cells(3),
		},
	}))

	assertScrollableListContains(t, text, "item 0")
	assertScrollableListContains(t, text, "item 1")
	assertScrollableListContains(t, text, "item 2")
	assertScrollableListNotContains(t, text, "item 3")
	assertScrollableListNotContains(t, text, "item 11")
}

func TestScrollableListEnterActivatesOrFallsBackToSelect(t *testing.T) {
	activated := -1
	selected := -1
	rt := runtime.New()
	root := ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(3),
		SelectedIndex: 2,
		AutoFocus:     true,
		OnSelect:      func(i int) { selected = i },
		OnActivate:    func(i int) { activated = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnter},
	})
	if !consumed {
		t.Fatal("Enter should be consumed")
	}
	if activated != 2 {
		t.Fatalf("OnActivate = %d, want 2", activated)
	}
	if selected != -1 {
		t.Fatalf("OnSelect should not be called when OnActivate exists, got %d", selected)
	}

	selected = -1
	rt = runtime.New()
	root = ScrollableList(ScrollableListProps{
		Items:         scrollableListTestItems(3),
		SelectedIndex: 1,
		AutoFocus:     true,
		OnSelect:      func(i int) { selected = i },
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed = rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnter},
	})
	if !consumed {
		t.Fatal("Enter should fall back to OnSelect")
	}
	if selected != 1 {
		t.Fatalf("OnSelect = %d, want 1", selected)
	}
}

func TestScrollableListFooterReservesRow(t *testing.T) {
	text := renderScrollableListText(ScrollableList(ScrollableListProps{
		Items:       scrollableListTestItems(6),
		ShowFooter:  true,
		VisibleRows: 3,
	}))

	assertScrollableListContains(t, text, "item 0")
	assertScrollableListContains(t, text, "item 1")
	assertScrollableListContains(t, text, "item 2")
	assertScrollableListNotContains(t, text, "item 3")
	assertScrollableListContains(t, text, "showing 1-3 of 6")
}

func TestScrollableListEmptyText(t *testing.T) {
	text := renderScrollableListText(ScrollableList(ScrollableListProps{}))
	assertScrollableListContains(t, text, "No items.")

	text = renderScrollableListText(ScrollableList(ScrollableListProps{
		EmptyText: "Nothing here.",
	}))
	assertScrollableListContains(t, text, "Nothing here.")
}

func renderScrollableListText(root Node) string {
	return renderScrollableListTextAt(root, 80, 24)
}

func renderScrollableListTextAt(root Node, w, h int) string {
	rt := runtime.New()
	updateUntilClean(rt, root)
	rt.RunLayout(w, h)
	buf := screen.NewBuffer(w, h)
	rt.Render(buf)
	return bufferText(buf)
}

func scrollableListTestItems(count int) []ScrollableListItem {
	items := make([]ScrollableListItem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, ScrollableListItem{
			ID:    fmt.Sprintf("item-%d", i),
			Label: fmt.Sprintf("item %d", i),
		})
	}
	return items
}

func assertScrollableListContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("rendered text does not contain %q in %q", want, text)
	}
}

func assertScrollableListNotContains(t *testing.T, text, want string) {
	t.Helper()
	if strings.Contains(text, want) {
		t.Fatalf("rendered text contains %q in %q", want, text)
	}
}
