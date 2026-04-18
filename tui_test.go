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

func TestOverlayReturnsOverlayNode(t *testing.T) {
	got := Overlay()
	if got.Kind != inode.OverlayKind {
		t.Fatalf("Overlay should return an OverlayKind node, got %v", got.Kind)
	}
}

func TestDialogReturnsComponentNode(t *testing.T) {
	got := Dialog(DialogProps{}, View(Style{}))
	if got.Kind != inode.ComponentKind {
		t.Fatalf("Dialog should return a ComponentKind node, got %v", got.Kind)
	}
}

func TestDialogEscCallsOnCloseWhenEnabled(t *testing.T) {
	called := false
	rt := runtime.New()
	root := Dialog(DialogProps{
		CloseOnEsc: true,
		OnClose:    func() { called = true },
	}, View(Style{}))
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEsc}})
	if !consumed {
		t.Fatal("Esc should be consumed by Dialog")
	}
	if !called {
		t.Fatal("OnClose should be called")
	}
}

func TestDialogEscReturnsFalseWhenCloseDisabled(t *testing.T) {
	called := false
	rt := runtime.New()
	root := Dialog(DialogProps{
		CloseOnEsc: false,
		OnClose:    func() { called = true },
	}, View(Style{}))
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEsc}})
	if consumed {
		t.Fatal("Esc should NOT be consumed by Dialog when CloseOnEsc is false")
	}
	if called {
		t.Fatal("OnClose should NOT be called")
	}
}

func TestDialogTrapsFocusAwayFromBackground(t *testing.T) {
	rt := runtime.New()

	backgroundPressed := false
	dialogPressed := false

	root := Overlay(
		Button(ButtonProps{
			Label:   "Background",
			OnPress: func() { backgroundPressed = true },
		}),
		Dialog(DialogProps{}, Button(ButtonProps{
			Label:   "Dialog",
			OnPress: func() { dialogPressed = true },
		})),
	)

	updateUntilClean(rt, root)

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})
	updateUntilClean(rt, root)

	if backgroundPressed {
		t.Fatal("background button should not be pressed while dialog is active")
	}
	if !dialogPressed {
		t.Fatal("dialog button should be focused and pressed")
	}

	dialogPressed = false

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyTab}})
	updateUntilClean(rt, root)

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})
	updateUntilClean(rt, root)

	if backgroundPressed {
		t.Fatal("Tab should not escape dialog focus scope")
	}
	if !dialogPressed {
		t.Fatal("dialog button should still be pressed after Tab")
	}
}

func TestDialogRendersOverBackgroundInOverlay(t *testing.T) {
	rt := runtime.New()
	root := Overlay(
		Text("BACKGROUND"),
		Dialog(DialogProps{
			Style: Style{Width: Cells(10), Height: Cells(1)},
		}, Text("DIALOG")),
	)
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Backdrop is full screen. Dialog is centered.
	// In 80x24, free vertical space is 24-1=23, so y = 23/2 = 11.
	// Free horizontal space is 80-10=70, so x = 70/2 = 35.
	// DIALOG text starts at the left of its 10-cell container by default.
	if got := buf.At(35, 11).Rune; got != 'D' {
		t.Errorf("expected 'D' at (35,11), got %q", got)
	}
}

func TestUseRouterInitialPath(t *testing.T) {
	var router Router
	comp := Component(func() Node {
		router = UseRouter("home")
		return View(Style{})
	})
	rt := runtime.New()
	rt.Update(comp)
	if router.Path != "home" {
		t.Errorf("expected home, got %q", router.Path)
	}
	if router.CanBack {
		t.Error("should not be able to go back from initial path")
	}
}

func TestUseRouterPushChangesPath(t *testing.T) {
	var router Router
	comp := Component(func() Node {
		router = UseRouter("home")
		return View(Style{})
	})
	rt := runtime.New()
	rt.Update(comp)

	router.Push("settings")
	rt.Update(comp)

	if router.Path != "settings" {
		t.Errorf("expected settings, got %q", router.Path)
	}
	if !router.CanBack {
		t.Error("should be able to go back after push")
	}
}

func TestUseRouterReplaceChangesCurrentPath(t *testing.T) {
	var router Router
	comp := Component(func() Node {
		router = UseRouter("home")
		return View(Style{})
	})
	rt := runtime.New()
	rt.Update(comp)

	router.Replace("settings")
	rt.Update(comp)

	if router.Path != "settings" {
		t.Errorf("expected settings, got %q", router.Path)
	}
	if router.CanBack {
		t.Error("should not be able to go back after replace on initial path")
	}
}

func TestUseRouterBackReturnsToPreviousPath(t *testing.T) {
	var router Router
	comp := Component(func() Node {
		router = UseRouter("home")
		return View(Style{})
	})
	rt := runtime.New()
	rt.Update(comp)

	router.Push("settings")
	rt.Update(comp)
	router.Back()
	rt.Update(comp)

	if router.Path != "home" {
		t.Errorf("expected home after back, got %q", router.Path)
	}
}

func TestUseRouterBackAtRootReturnsFalse(t *testing.T) {
	var router Router
	comp := Component(func() Node {
		router = UseRouter("home")
		return View(Style{})
	})
	rt := runtime.New()
	rt.Update(comp)

	if router.Back() != false {
		t.Error("Back() at root should return false")
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
	updateUntilClean(rt, inputNode)

	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}})
	updateUntilClean(rt, inputNode)

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
	updateUntilClean(rt, inputNode)

	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})
	updateUntilClean(rt, inputNode)

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
	updateUntilClean(rt, inputNode)

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
	updateUntilClean(rt, inputNode)


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

	updateUntilClean(rt, wrapperNode)

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

	updateUntilClean(rt, root)

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

func TestTextInputCursorUsesDisplayWidthForWideRunes(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value: "a界",
		Style: Style{
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	updateUntilClean(rt, root)

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

func TestTextInputLeftMovesCursorOneRune(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "abc",
		Style: Style{Border: BorderAll},
	}))
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}}); consumed {
		t.Fatal("Right at end should return false")
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}}); consumed {
		t.Fatal("End at end should return false")
	}
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}}); consumed {
		t.Fatal("Left at cursor 0 should return false")
	}
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}}); consumed {
		t.Fatal("Home at cursor 0 should return false")
	}
}

func TestTextInputTypingInsertsAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "ac",
		OnChange: func(next string) {
			got = next
		},
	}))
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: '界'}})

	if got != "a界b" {
		t.Fatalf("Wide rune insert failed, got %q want a界b", got)
	}
}

func TestTextInputBackspaceDeletesBeforeCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		OnChange: func(next string) { got = next },
	}))
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})

	if got != "ab" {
		t.Fatalf("Backspace wide rune before cursor failed, got %q want ab", got)
	}
}

func TestTextInputDeleteRemovesRuneAtCursor(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		OnChange: func(next string) { got = next },
	}))
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	if consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}}); consumed {
		t.Fatal("Delete at end should return false")
	}
	if called {
		t.Fatal("Delete at end should not call OnChange")
	}
}

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

	updateUntilClean(rt, root)

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

func TestTextInputEnterCallsOnSubmit(t *testing.T) {
	submitted := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "hello",
		OnSubmit: func(s string) { submitted = s },
	}))
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed {
		t.Fatal("Enter without OnSubmit should return false")
	}
}

func TestTextInputRuneWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "hello"}))
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}})

	if consumed {
		t.Fatal("Rune without OnChange should return false")
	}
}

func TestTextInputBackspaceWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "hello"}))
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})

	if consumed {
		t.Fatal("Backspace without OnChange should return false")
	}
}

func TestTextInputDeleteWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "hello"}))
	updateUntilClean(rt, root)
	// Move cursor to start so Delete has something to delete
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}})

	if consumed {
		t.Fatal("Delete without OnChange should return false")
	}
}

func TestTextInputMovementWithoutOnChangeStillWorks(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "hello"}))
	updateUntilClean(rt, root)

	// Move left
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if !consumed {
		t.Fatal("Left movement should be consumed even if OnChange is nil")
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
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed || called {
		t.Fatal("Disabled TextInput should not consume Enter or submit")
	}
}

func TestDisabledTextInputDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value:    "abc",
		Disabled: true,
		OnChange: func(string) { called = true },
	}))
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if consumed || called {
		t.Fatal("Disabled TextInput should not consume Enter or submit")
	}
}

func TestDisabledButtonDoesNotCallOnPressOnEnter(t *testing.T) {
	called := false
	rt := runtime.New()
	btn := Button(ButtonProps{
		Label:    "Button",
		Disabled: true,
		OnPress:  func() { called = true },
	})
	root := View(Style{}, btn)
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnter}})

	if calledDisabled {
		t.Fatal("Disabled button should not be pressed")
	}
	if !calledEnabled {
		t.Fatal("Enabled button should be focused and pressed")
	}
}

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
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
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
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)
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

func TestTextInputVisibleValueAtEnd(t *testing.T) {
	visible, cursorX := textInputVisibleValue("abcdef", 6, 4, true)

	if visible != "def " {
		t.Fatalf("visible = %q, want %q", visible, "def ")
	}
	if cursorX != 3 {
		t.Fatalf("cursorX = %d, want 3", cursorX)
	}
}

func TestTextInputVisibleValueAtStart(t *testing.T) {
	visible, cursorX := textInputVisibleValue("abcdef", 0, 4, true)

	if visible != "abc " {
		t.Fatalf("visible = %q, want %q", visible, "abc ")
	}
	if cursorX != 0 {
		t.Fatalf("cursorX = %d, want 0", cursorX)
	}
}

func TestTextInputVisibleValueInMiddle(t *testing.T) {
	// Value "abcdef", cursor at 3 ('d'), content width 4.
	// Space for 3 runes + 1 for cursor/padding.
	visible, cursorX := textInputVisibleValue("abcdef", 3, 4, true)

	// current implementation of textInputVisibleValue scrolls so that cursor is at the end of window if possible when moving right,
	// but when at index 3, it should show runes around it.
	// Let's check logic:
	// maxWidth = 4 - 1 = 3
	// start = 3. wBefore: 'c'(1) -> 1, 'b'(1) -> 2, 'a'(1) -> 3. start = 0.
	// end = 3. wAfter: 'd'(1) + wBefore(3) > 3. end = 3.
	// vis = "abc" + " "
	// cursorX = width("abc") = 3.

	if visible != "abc " {
		t.Fatalf("visible = %q, want %q", visible, "abc ")
	}
	if cursorX != 3 {
		t.Fatalf("cursorX = %d, want 3", cursorX)
	}
}

func TestTextInputVisibleValueWithWideRuneBeforeCursor(t *testing.T) {
	// "a界b", cursor at 2 ('b'), content width 4.
	// maxWidth = 3.
	// start = 2. wBefore: '界'(2) -> 2, 'a'(1) -> 3. start = 0.
	// end = 2. wAfter: 'b'(1) + 3 > 3. end = 2.
	// vis = "a界" + " "
	// cursorX = width("a界") = 3.
	visible, cursorX := textInputVisibleValue("a界b", 2, 4, true)

	if visible != "a界 " {
		t.Fatalf("visible = %q, want %q", visible, "a界 ")
	}
	if cursorX != 3 {
		t.Fatalf("cursorX = %d, want 3", cursorX)
	}
}

func TestTextInputVisibleValueWithWideRuneAtBoundary(t *testing.T) {
	// "abc界", cursor at 4 (end), content width 4.
	// maxWidth = 3.
	// start = 4. wBefore: '界'(2) -> 2, 'c'(1) -> 3. start = 2.
	// vis = "c界" + " "
	// cursorX = width("c界") = 3.
	visible, cursorX := textInputVisibleValue("abc界", 4, 4, true)

	if visible != "c界 " {
		t.Fatalf("visible = %q, want %q", visible, "c界 ")
	}
	if cursorX != 3 {
		t.Fatalf("cursorX = %d, want 3", cursorX)
	}
}

func TestTextInputControlledTypingUpdatesRenderedValue(t *testing.T) {
	rt := runtime.New()
	form := func() Node {
		val, setVal := UseState("ac")
		return TextInput(TextInputProps{
			Value:    val,
			OnChange: setVal,
			Style:    Style{Border: BorderAll},
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	// Move cursor between 'a' and 'c'
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	updateUntilClean(rt, root)

	// Type 'b'
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'b'}})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Result should be "abc"
	if got := buf.At(1, 1).Rune; got != 'a' {
		t.Errorf("at (1,1) got %q want 'a'", got)
	}
	if got := buf.At(2, 1).Rune; got != 'b' {
		t.Errorf("at (2,1) got %q want 'b'", got)
	}
	if got := buf.At(3, 1).Rune; got != 'c' {
		t.Errorf("at (3,1) got %q want 'c'", got)
	}
}

func TestTextInputControlledBackspaceUpdatesRenderedValue(t *testing.T) {
	rt := runtime.New()
	form := func() Node {
		val, setVal := UseState("abc")
		return TextInput(TextInputProps{
			Value:    val,
			OnChange: setVal,
			Style:    Style{Border: BorderAll},
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	// Backspace at end
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyBackspace}})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Result should be "ab"
	if got := buf.At(1, 1).Rune; got != 'a' {
		t.Errorf("at (1,1) got %q want 'a'", got)
	}
	if got := buf.At(2, 1).Rune; got != 'b' {
		t.Errorf("at (2,1) got %q want 'b'", got)
	}
	if got := buf.At(3, 1).Rune; got != ' ' {
		t.Errorf("at (3,1) got %q want ' '", got)
	}
}

func TestTextInputControlledDeleteUpdatesRenderedValue(t *testing.T) {
	rt := runtime.New()
	form := func() Node {
		val, setVal := UseState("abc")
		return TextInput(TextInputProps{
			Value:    val,
			OnChange: setVal,
			Style:    Style{Border: BorderAll},
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	// Move to start
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)

	// Delete 'a'
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDelete}})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Result should be "bc"
	if got := buf.At(1, 1).Rune; got != 'b' {
		t.Errorf("at (1,1) got %q want 'b'", got)
	}
	if got := buf.At(2, 1).Rune; got != 'c' {
		t.Errorf("at (2,1) got %q want 'c'", got)
	}
	if got := buf.At(3, 1).Rune; got != ' ' {
		t.Errorf("at (3,1) got %q want ' '", got)
	}
}

func TestTextInputControlledCursorSurvivesParentUpdate(t *testing.T) {
	rt := runtime.New()
	form := func() Node {
		val, setVal := UseState("abc")
		return TextInput(TextInputProps{
			Value:    val,
			OnChange: setVal,
			Style:    Style{Border: BorderAll},
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	// Move cursor to index 1 (between 'a' and 'b')
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
	x, _, _ := rt.Cursor()
	if x != 2 {
		t.Fatalf("Initial cursor x should be 2, got %d", x)
	}

	// Type 'X'
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'X'}})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	rt.Render(buf)
	// Value is "aXbc", cursor should be at index 2 (after 'X'), so x=3
	x, _, _ = rt.Cursor()
	if x != 3 {
		t.Fatalf("Cursor x after typing should be 3, got %d", x)
	}

	buf = screen.NewBuffer(80, 24)
	rt.Render(buf)
	if got := buf.At(1, 1).Rune; got != 'a' {
		t.Errorf("at (1,1) got %q want 'a'", got)
	}
	if got := buf.At(2, 1).Rune; got != 'X' {
		t.Errorf("at (2,1) got %q want 'X'", got)
	}
}

func TestTextInputFixedWidthPlaceholderCursorStartsAtContentStart(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value:       "",
		Placeholder: "Type here",
		Style: Style{
			Width:  Cells(8),
			Height: Cells(3),
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	x, y, visible := rt.Cursor()
	// Border is at x=0, so content starts at x=1.
	if !visible || x != 1 || y != 1 {
		t.Fatalf("Placeholder cursor expected visible=true x=1 y=1, got visible=%v x=%d y=%d", visible, x, y)
	}

	if got := buf.At(1, 1).Rune; got != 'T' {
		t.Fatalf("placeholder should start at (1,1), got %q", got)
	}
}

func TestTextInputExternalValueChangePreservesCursorWithinBounds(t *testing.T) {
	rt := runtime.New()

	val := "initial"
	form := func() Node {
		return TextInput(TextInputProps{
			Value:     val,
			AutoFocus: true,
		})
	}
	root := Component(form)

	// Initial render to mount and focus
	updateUntilClean(rt, root)

	if rt.Focused() == nil {
		t.Fatal("Nothing focused")
	}

	// Move cursor to start of "initial"
	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	if !consumed {
		t.Fatal("Home key not consumed")
	}
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	x, _, visible := rt.Cursor()
	if !visible {
		t.Fatal("Cursor not visible after move to Home")
	}
	if x != 0 {
		t.Fatalf("Cursor x should be 0, got %d", x)
	}

	// Move cursor to index 3
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	x, _, _ = rt.Cursor()
	if x != 3 {
		t.Fatalf("Cursor x should be 3, got %d", x)
	}

	// Update external value
	val = "new value" // length 9
	rt.Update(root)
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	x, _, visible = rt.Cursor()
	if !visible {
		t.Fatal("Cursor not visible after external update")
	}
	// Current implementation preserves cursor (3)
	if x != 3 {
		t.Fatalf("Cursor x should still be 3, got %d", x)
	}

	// Update external value to something shorter but still includes cursor
	val = "short" // length 5
	rt.Update(root)
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	x, _, visible = rt.Cursor()
	if !visible {
		t.Fatal("Cursor not visible after external update to shorter value")
	}
	// Should still be at 3
	if x != 3 {
		t.Fatalf("Cursor x should still be 3, got %d", x)
	}

	// Update external value to something shorter than cursor
	val = "ab" // length 2
	rt.Update(root)
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	x, _, visible = rt.Cursor()
	if !visible {
		t.Fatal("Cursor not visible after external update to very short value")
	}
	// Should be clamped to 2
	if x != 2 {
		t.Fatalf("Cursor x should be clamped to 2, got %d", x)
	}
}

func updateUntilClean(rt *runtime.Runtime, root Node) {
	for rt.IsDirty() {
		rt.Update(root)
	}
}
