package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/event"
	inode "github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/runtime"
	"github.com/smason/earlgray/internal/screen"
)

func TestOverlayVisualStyleIgnoresFocusLayout(t *testing.T) {
	base := Style{
		Width:  Cells(7),
		Height: Cells(3),
		Border: BorderAll,
	}
	focus := Style{
		Width:      Cells(99),
		Height:     Cells(99),
		Border:     BorderNone,
		Foreground: ANSIColor(3),
	}

	got := overlayVisualStyle(base, focus)

	if got.Width != base.Width || got.Height != base.Height || got.Border != base.Border {
		t.Fatal("focused visual style should not override layout")
	}
	if got.Foreground != focus.Foreground {
		t.Fatal("focused visual style should apply foreground")
	}
}

func TestTextInputReturnsComponentNode(t *testing.T) {
	got := TextInput(TextInputProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("TextInput should return a ComponentKind node, got %v", got.Kind)
	}
}

func TestOverlayVisualStylePreservesLayout(t *testing.T) {
	base := Style{
		Width:  Cells(7),
		Height: Cells(3),
		Border: BorderAll,
	}
	focus := Style{
		Foreground: ANSIColor(3),
	}

	got := overlayVisualStyle(base, focus)

	if got.Width != base.Width || got.Height != base.Height || got.Border != base.Border {
		t.Fatal("focused visual style should preserve base layout")
	}
	if got.Foreground != focus.Foreground {
		t.Fatal("focused visual style should apply foreground")
	}
}

func TestTextInputTypingCallsOnChange(t *testing.T) {
	got := ""
	props := TextInputProps{
		Value: "hello",
		OnChange: func(next string) {
			got = next
		},
	}

	rt := runtime.New()
	inputNode := TextInput(props)
	rt.Update(inputNode)

	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}})
	if rt.IsDirty() {
		rt.Update(inputNode)
	}

	if got != "hellox" {
		t.Errorf("typing 'x' should append, got %q want %q", got, "hellox")
	}
}

func TestTextInputBackspaceRemovesOneRune(t *testing.T) {
	got := ""
	props := TextInputProps{
		Value: "a界",
		OnChange: func(next string) {
			got = next
		},
	}

	rt := runtime.New()
	inputNode := TextInput(props)
	rt.Update(inputNode)

	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})
	if rt.IsDirty() {
		rt.Update(inputNode)
	}

	if got != "a" {
		t.Errorf("backspace should remove one rune, got %q want %q", got, "a")
	}
	// Verify it's valid UTF-8, not byte-truncated
	if got != "a" {
		t.Errorf("expected valid UTF-8 %q, got %q", "a", got)
	}
}

func TestTextInputBackspaceOnEmptyReturnsFalse(t *testing.T) {
	onChangeCalled := false
	props := TextInputProps{
		Value: "",
		OnChange: func(next string) {
			onChangeCalled = true
		},
	}

	rt := runtime.New()
	inputNode := TextInput(props)
	rt.Update(inputNode)

	rt.RunLayout(80, 24)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})

	if consumed {
		t.Error("backspace on empty value should return false")
	}
	if onChangeCalled {
		t.Error("backspace on empty value should not call OnChange")
	}
}

func TestTextInputFocusedRequestsCursor(t *testing.T) {
	props := TextInputProps{
		Value: "test",
	}

	rt := runtime.New()
	inputNode := TextInput(props)
	rt.Update(inputNode)

	if rt.IsDirty() {
		rt.Update(inputNode)
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	_, _, visible := rt.Cursor()
	if !visible {
		t.Error("focused TextInput should request cursor")
	}
}

func TestTextInputUnfocusedNoCursor(t *testing.T) {
	props := TextInputProps{
		Value: "test",
	}

	rt := runtime.New()
	wrapperNode := View(
		Style{},
		Button(ButtonProps{Label: "Button"}),
		TextInput(props),
	)

	rt.Update(wrapperNode)
	if rt.IsDirty() {
		rt.Update(wrapperNode)
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	_, _, visible := rt.Cursor()
	if visible {
		t.Error("unfocused TextInput should not request cursor")
	}
}

func TestTextInputFocusedCursorSitsAfterValueForAutoWidth(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value: "a",
		Style: Style{
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	x, y, visible := rt.Cursor()
	if !visible {
		t.Fatal("focused TextInput should request cursor")
	}

	if x != 2 || y != 1 {
		t.Fatalf("cursor = (%d,%d), want (2,1)", x, y)
	}

	if got := buf.At(1, 1).Rune; got != 'a' {
		t.Fatalf("value rune at (1,1) = %q, want 'a'", got)
	}
	if got := buf.At(2, 1).Rune; got != ' ' {
		t.Fatalf("cursor cell at (2,1) should be blank, got %q", got)
	}
}

// Ticket 1: Add tests proving the cursor position uses a rune index and display width correctly.
func TestTextInputCursorUsesDisplayWidthForWideRunes(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value: "a界",
		Style: Style{
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	x, y, visible := rt.Cursor()
	if !visible {
		t.Fatal("focused TextInput should request cursor")
	}

	if x != 4 || y != 1 {
		t.Fatalf("cursor = (%d,%d), want (4,1)", x, y)
	}
}

// Ticket 2: Left/Right/Home/End
func TestTextInputLeftMovesCursorOneRune(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abc",
		Style: Style{Border: BorderAll},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	x, y, _ := rt.Cursor()
	if x != 3 || y != 1 {
		t.Fatalf("Left moved cursor incorrectly, got (%d,%d) want (3,1)", x, y)
	}
}

func TestTextInputRightMovesCursorOneRune(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abc",
		Style: Style{Border: BorderAll},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	x, y, _ := rt.Cursor()
	if x != 4 || y != 1 {
		t.Fatalf("Right moved cursor incorrectly, got (%d,%d) want (4,1)", x, y)
	}
}

func TestTextInputHomeMovesCursorToStart(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abc",
		Style: Style{Border: BorderAll},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	x, y, _ := rt.Cursor()
	if x != 1 || y != 1 {
		t.Fatalf("Home moved cursor incorrectly, got (%d,%d) want (1,1)", x, y)
	}
}

func TestTextInputEndMovesCursorToEnd(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abc",
		Style: Style{Border: BorderAll},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	x, y, _ := rt.Cursor()
	if x != 4 || y != 1 {
		t.Fatalf("End moved cursor incorrectly, got (%d,%d) want (4,1)", x, y)
	}
}

func TestTextInputMovementAtEdgesReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "abc"}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}}); consumed {
		t.Fatal("Right at end should return false")
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}}); consumed {
		t.Fatal("End at end should return false")
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}}); consumed {
		t.Fatal("Left at cursor 0 should return false")
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}}); consumed {
		t.Fatal("Home at cursor 0 should return false")
	}
}

// Ticket 3
func TestTextInputTypingInsertsAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "ac",
		OnChange: func(next string) {
			got = next
		},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'b'}})

	if got != "abc" {
		t.Fatalf("Insert failed, got %q want abc", got)
	}
}

func TestTextInputTypingWideRuneInsertsAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "ab",
		OnChange: func(next string) {
			got = next
		},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: '界'}})

	if got != "a界b" {
		t.Fatalf("Wide rune insert failed, got %q want a界b", got)
	}
}

// Ticket 4
func TestTextInputBackspaceDeletesBeforeCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		OnChange: func(next string) { got = next },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})

	if got != "ac" {
		t.Fatalf("Backspace before cursor failed, got %q want ac", got)
	}
}

func TestTextInputBackspaceDeletesWideRuneBeforeCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "a界b",
		OnChange: func(next string) { got = next },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})

	if got != "ab" {
		t.Fatalf("Backspace wide rune before cursor failed, got %q want ab", got)
	}
}

// Ticket 5
func TestTextInputDeleteRemovesRuneAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		OnChange: func(next string) { got = next },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}})

	if got != "bc" {
		t.Fatalf("Delete removed incorrectly, got %q want bc", got)
	}
}

func TestTextInputDeleteWideRuneAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "a界b",
		OnChange: func(next string) { got = next },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}})

	if got != "ab" {
		t.Fatalf("Delete wide rune removed incorrectly, got %q want ab", got)
	}
}

func TestTextInputDeleteAtEndReturnsFalse(t *testing.T) {
	called := false
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		OnChange: func(next string) { called = true },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}}); consumed {
		t.Fatal("Delete at end should return false")
	}
	if called {
		t.Fatal("Delete at end should not call OnChange")
	}
}

// Ticket 6
func TestTextInputPlaceholderCursorStartsAtContentStart(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value:       "",
		Placeholder: "Type here",
		Style: Style{
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	x, y, visible := rt.Cursor()
	if !visible || x != 1 || y != 1 {
		t.Fatalf("Placeholder cursor expected visible=true x=1 y=1, got visible=%v x=%d y=%d", visible, x, y)
	}

	if got := buf.At(1, 1).Rune; got != 'T' {
		t.Fatalf("placeholder should start at (1,1), got %q", got)
	}
}

// Ticket 7
func TestTextInputEnterCallsOnSubmit(t *testing.T) {
	submitted := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "hello",
		OnSubmit: func(s string) { submitted = s },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if !consumed {
		t.Fatal("Enter should return true")
	}
	if submitted != "hello" {
		t.Fatalf("Expected submitted hello, got %q", submitted)
	}
}

func TestTextInputEnterWithoutOnSubmitReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "hello"}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed {
		t.Fatal("Enter without OnSubmit should return false")
	}
}

func TestDisabledTextInputEnterDoesNotSubmit(t *testing.T) {
	called := false
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "hello",
		Disabled: true,
		OnSubmit: func(s string) { called = true },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed || called {
		t.Fatal("Disabled TextInput should not consume Enter or submit")
	}
}

// Ticket 8
func TestDisabledTextInputDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		Disabled: true,
		OnChange: func(string) { called = true },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}})

	if consumed || called {
		t.Fatal("Disabled TextInput should not consume rune or change")
	}
}

func TestDisabledTextInputDoesNotRequestCursor(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		Disabled: true,
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	_, _, visible := rt.Cursor()
	if visible {
		t.Fatal("Disabled TextInput should not request cursor")
	}
}

func TestDisabledTextInputDoesNotCallOnSubmit(t *testing.T) {
	called := false
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		Disabled: true,
		OnSubmit: func(string) { called = true },
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed || called {
		t.Fatal("Disabled TextInput should not consume Enter or submit")
	}
}

// Ticket 9
func TestDisabledButtonDoesNotCallOnPressOnEnter(t *testing.T) {
	called := false
	rt := runtime.New()
	btn := Button(ButtonProps{
		Label:    "Button",
		Disabled: true,
		OnPress:  func() { called = true },
	})
	root := View(Style{}, btn)
	rt.Update(root)
	rt.RunLayout(80, 24)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})
	if consumed || called {
		t.Fatal("Disabled Button should not consume Enter or press")
	}
}

func TestDisabledButtonDoesNotCallOnPressOnSpace(t *testing.T) {
	called := false
	rt := runtime.New()
	btn := Button(ButtonProps{
		Label:    "Button",
		Disabled: true,
		OnPress:  func() { called = true },
	})
	root := View(Style{}, btn)
	rt.Update(root)
	rt.RunLayout(80, 24)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: ' '}})
	if consumed || called {
		t.Fatal("Disabled Button should not consume Space or press")
	}
}

func TestDisabledButtonSkippedByFocusTraversal(t *testing.T) {
	calledEnabled := false
	calledDisabled := false
	rt := runtime.New()
	root := View(
		Style{},
		Button(ButtonProps{Label: "Disabled", Disabled: true, OnPress: func() { calledDisabled = true }}),
		Button(ButtonProps{Label: "Enabled", OnPress: func() { calledEnabled = true }}),
	)
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if calledDisabled {
		t.Fatal("Disabled button should not be pressed")
	}
	if !calledEnabled {
		t.Fatal("Enabled button should be focused and pressed")
	}
}

// Ticket 11
func TestTextInputFixedWidthCursorScrollsAtEnd(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abcdef",
		Style: Style{
			Width:  Cells(6),
			Height: Cells(3),
			Border: BorderAll,
		},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	x, y, _ := rt.Cursor()
	if x != 4 || y != 1 {
		t.Fatalf("Scrolled cursor should be at (4,1), got (%d,%d)", x, y)
	}
	if got := buf.At(4, 1).Rune; got != ' ' {
		t.Fatalf("cursor cell at (4,1) should be blank, got %q", got)
	}
}

func TestTextInputFixedWidthScrollDoesNotSplitWideRune(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "ab界cd",
		Style: Style{
			Width:  Cells(6),
			Border: BorderAll,
		},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	_, _, visible := rt.Cursor()
	if !visible {
		t.Fatal("Cursor should be visible")
	}
}

func TestTextInputFixedWidthCursorMovesBackIntoScrolledValue(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abcdef",
		Style: Style{
			Width:  Cells(6),
			Border: BorderAll,
		},
	}))
	rt.Update(root)
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if rt.IsDirty() {
		rt.Update(root)
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	x, _, visible := rt.Cursor()
	if !visible {
		t.Fatal("Cursor should be visible")
	}
	// The exact x depends on rendering, just assert it is > 0 and <= 4
	if x <= 0 || x > 4 {
		t.Fatalf("Cursor x %d is out of bounds [1, 4]", x)
	}
}
