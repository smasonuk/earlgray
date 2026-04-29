package tui

import (
	"testing"

	"github.com/smasonuk/earlgray/internal/event"
	"github.com/smasonuk/earlgray/internal/host"
)

func TestTextInputMouseCursorPlacement(t *testing.T) {
	app := func() Node {
		return TextInput(TextInputProps{
			Value:     "hello",
			AutoFocus: true,
		})
	}

	// Initial cursor is at end (index 5, x=5)
	// Click at x=2 should move cursor to index 2, x=2
	events := []event.Event{
		{
			Kind: event.MouseKind,
			Mouse: event.Mouse{
				X:      2,
				Y:      0,
				Button: MouseLeft,
			},
		},
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if fake.cursorX != 2 {
		t.Errorf("expected cursorX=2, got %d", fake.cursorX)
	}
}

func TestTextInputMouseCursorPlacementWideRunes(t *testing.T) {
	app := func() Node {
		return TextInput(TextInputProps{
			Value:     "世界",
			AutoFocus: true,
		})
	}

	// Click at x=1 (middle of "世") -> index 1 (after "世" cells)
	events := []event.Event{
		{
			Kind: event.MouseKind,
			Mouse: event.Mouse{
				X:      1,
				Y:      0,
				Button: MouseLeft,
			},
		},
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if fake.cursorX != 2 { // "世" is 2 cells wide
		t.Errorf("expected cursorX=2, got %d", fake.cursorX)
	}
}

func TestListMouseClickFocusAndSelect(t *testing.T) {
	selected := -1
	app := func() Node {
		return List(ListProps{
			Items:    []string{"A", "B", "C"},
			OnSelect: func(i int) { selected = i },
		})
	}

	// Click on second item "B" at (0, 1)
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

	if selected != 1 {
		t.Errorf("expected selected=1, got %d", selected)
	}
}

func TestRadioGroupMouseClickFocusAndSelect(t *testing.T) {
	selected := ""
	app := func() Node {
		return RadioGroup(RadioGroupProps{
			Options: []RadioOption{
				{Label: "One", Value: "1"},
				{Label: "Two", Value: "2"},
			},
			Value:    "1",
			OnChange: func(v string) { selected = v },
		})
	}

	// Click on second option "Two" at (0, 1)
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

	if selected != "2" {
		t.Errorf("expected selected='2', got %q", selected)
	}
}

func TestSideTabsMouseClickSelectsTab(t *testing.T) {
	selected := ""
	app := func() Node {
		return SideTabs(SideTabsProps{
			Value: "home",
			OnChange: func(v string) {
				selected = v
			},
			Tabs: []SideTab{
				{Label: "Home", Value: "home", Content: Text("HOME")},
				{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
			},
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

	if selected != "settings" {
		t.Errorf("expected selected='settings', got %q", selected)
	}
}

func TestSideTabsMouseClickDisabledTabDoesNothing(t *testing.T) {
	selected := ""
	app := func() Node {
		return SideTabs(SideTabsProps{
			Value: "home",
			OnChange: func(v string) {
				selected = v
			},
			Tabs: []SideTab{
				{Label: "Home", Value: "home", Content: Text("HOME")},
				{Label: "Settings", Value: "settings", Content: Text("SETTINGS"), Disabled: true},
			},
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

	if selected != "" {
		t.Errorf("expected no selection for disabled tab, got %q", selected)
	}
}

func TestDisabledSideTabsMouseClickDoesNothing(t *testing.T) {
	selected := ""
	app := func() Node {
		return SideTabs(SideTabsProps{
			Value: "home",
			OnChange: func(v string) {
				selected = v
			},
			Disabled: true,
			Tabs: []SideTab{
				{Label: "Home", Value: "home", Content: Text("HOME")},
				{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
			},
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

	if selected != "" {
		t.Errorf("expected no selection for disabled SideTabs, got %q", selected)
	}
}

// TestTextInputUnfocusedFixedCursorAtStartOnClick verifies Ticket 2 fix:
// when a fixed-width TextInput is unfocused, clicking near the start of the
// visible area places the cursor at the beginning of the string, not near
// the end where the internal cursor was previously positioned.
func TestTextInputUnfocusedFixedCursorAtStartOnClick(t *testing.T) {
	// Button at y=0 gets AutoFocus; TextInput at y=1 starts unfocused with cursor=26.
	app := func() Node {
		return View(Style{Direction: Column},
			Button(ButtonProps{Label: "btn", AutoFocus: true}),
			Keyed("input", TextInput(TextInputProps{
				Value: "abcdefghijklmnopqrstuvwxyz",
				Style: Style{Width: Cells(10)},
			})),
		)
	}
	events := []event.Event{
		{Kind: event.MouseKind, Mouse: event.Mouse{X: 0, Y: 1, Button: MouseLeft}},
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}
	// With fix: start=0 (unfocused), click at LocalX=0 → cursor=0, CursorX=0.
	// Without fix: start=17 (scrolled from cursor=26), click at LocalX=0 → cursor=17, CursorX=9.
	if fake.cursorX != 0 {
		t.Errorf("unfocused fixed TextInput click at start: expected cursorX=0, got %d", fake.cursorX)
	}
}

// TestTextInputFocusedFixedScrolledCursorOnClick verifies that the fix does not
// break focused fixed-width inputs: clicking inside a scrolled focused input
// still maps against the currently visible scrolled substring.
func TestTextInputFocusedFixedScrolledCursorOnClick(t *testing.T) {
	// TextInput focused with cursor at end (index 26). Width=10, maxWidth=9.
	// Visible window starts at rune 17. Click at x=0 (LocalX=0) → cursor=17.
	// After re-render: visible window backs up 9 from 17 → start=8, CursorX=9.
	app := func() Node {
		return TextInput(TextInputProps{
			Value:     "abcdefghijklmnopqrstuvwxyz",
			AutoFocus: true,
			Style:     Style{Width: Cells(10)},
		})
	}
	events := []event.Event{
		{Kind: event.MouseKind, Mouse: event.Mouse{X: 0, Y: 0, Button: MouseLeft}},
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}
	if fake.cursorX != 9 {
		t.Errorf("focused scrolled TextInput click at left edge: expected cursorX=9, got %d", fake.cursorX)
	}
}

func TestDisabledListMouseClick(t *testing.T) {
	selected := -1
	app := func() Node {
		return List(ListProps{
			Items:    []string{"A", "B", "C"},
			OnSelect: func(i int) { selected = i },
			Disabled: true,
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

	if selected != -1 {
		t.Errorf("expected no selection for disabled list, got %d", selected)
	}
}
