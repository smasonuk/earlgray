package tui

import (
	"testing"

	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/host"
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
	if err := runWithHost(app, func() (host.Host, error) { return fake, nil }); err != nil {
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
	if err := runWithHost(app, func() (host.Host, error) { return fake, nil }); err != nil {
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
	if err := runWithHost(app, func() (host.Host, error) { return fake, nil }); err != nil {
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
	if err := runWithHost(app, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if selected != "2" {
		t.Errorf("expected selected='2', got %q", selected)
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
	if err := runWithHost(app, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if selected != -1 {
		t.Errorf("expected no selection for disabled list, got %d", selected)
	}
}
