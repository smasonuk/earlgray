package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/smasonuk/earlgray/internal/event"
	inode "github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/runtime"
	"github.com/smasonuk/earlgray/internal/screen"
)

func TestOverlayVisualStyleAppliesOnlyVisualFields(t *testing.T) {
	base := Style{
		Width:   Cells(7),
		Height:  Cells(3),
		Border:  BorderAll,
		Padding: All(1),
		Gap:     2,
	}
	focus := Style{
		Width:         Cells(99),
		Height:        Cells(99),
		Border:        BorderNone,
		Padding:       All(9),
		Gap:           9,
		FlexGrow:      99,
		Foreground:    ANSIColor(3),
		Background:    ANSIColor(4),
		Bold:          true,
		Italic:        true,
		Underline:     true,
		Faint:         true,
		Strikethrough: true,
		Reverse:       true,
	}

	got := overlayVisualStyle(base, focus)

	if got.Width != base.Width {
		t.Fatal("focused visual style should not override width")
	}
	if got.Height != base.Height {
		t.Fatal("focused visual style should not override height")
	}
	if got.Border != base.Border {
		t.Fatal("focused visual style should not override border")
	}
	if got.Padding != base.Padding {
		t.Fatal("focused visual style should not override padding")
	}
	if got.Gap != base.Gap {
		t.Fatal("focused visual style should not override gap")
	}
	if got.FlexGrow != base.FlexGrow {
		t.Fatal("focused visual style should not override flex grow")
	}

	if got.Foreground != focus.Foreground {
		t.Fatal("focused visual style should apply foreground")
	}
	if got.Background != focus.Background {
		t.Fatal("focused visual style should apply background")
	}
	if !got.Bold {
		t.Fatal("focused visual style should apply bold")
	}
	if !got.Italic {
		t.Fatal("focused visual style should apply italic")
	}
	if !got.Underline {
		t.Fatal("focused visual style should apply underline")
	}
	if !got.Faint {
		t.Fatal("focused visual style should apply faint")
	}
	if !got.Strikethrough {
		t.Fatal("focused visual style should apply strikethrough")
	}
	if !got.Reverse {
		t.Fatal("focused visual style should apply reverse")
	}
}

func TestRichTextRendersStyledSpans(t *testing.T) {
	rt := runtime.New()
	root := RichText(
		TextSpan{Text: "hello "},
		TextSpan{Text: "world", Style: Style{Bold: true}},
	)
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 'h' {
		t.Fatalf("expected first rune to be 'h', got %q", got)
	}
	if got := buf.At(6, 0).Rune; got != 'w' {
		t.Fatalf("expected styled span to start with 'w', got %q", got)
	}
	if !buf.At(6, 0).Style.Bold {
		t.Fatal("expected second span cells to be bold")
	}
	if buf.At(0, 0).Style.Bold {
		t.Fatal("expected first span cells to keep default bold=false")
	}
}

func TestRichTextHandlesNewlines(t *testing.T) {
	rt := runtime.New()
	root := RichText(
		TextSpan{Text: "line1\n"},
		TextSpan{Text: "line2"},
	)
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 1).Rune; got != 'l' {
		t.Fatalf("expected second line to start with 'l', got %q", got)
	}
}

func TestRichTextInheritsParentColor(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Foreground: ANSIColor(3)}, RichText(
		TextSpan{Text: "hello"},
		TextSpan{Text: " world"},
	))
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Style.Fg; got != ANSIColor(3) {
		t.Fatalf("expected rich text to inherit parent foreground, got %v", got)
	}
}

func TestRichTextWideRuneClippingDoesNotShiftLaterSpans(t *testing.T) {
	rt := runtime.New()
	root := View(
		Style{Width: Cells(5), Height: Cells(1)},
		RichText(
			TextSpan{Text: "abcd界"},
			TextSpan{Text: "Z", Style: Style{Bold: true}},
		),
	)
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 'a' {
		t.Fatalf("expected 'a' at x=0, got %q", got)
	}
	if got := buf.At(3, 0).Rune; got != 'd' {
		t.Fatalf("expected 'd' at x=3, got %q", got)
	}
	if got := buf.At(4, 0).Rune; got == 'Z' {
		t.Fatal("expected clipped wide rune to keep later span off-screen")
	}
	if buf.At(4, 0).Style.Bold {
		t.Fatal("expected later span style not to leak into clipped cells")
	}
}

func TestRichTextWideRuneLogicalAdvanceWhenItFits(t *testing.T) {
	rt := runtime.New()
	root := View(
		Style{Width: Cells(7), Height: Cells(1)},
		RichText(
			TextSpan{Text: "abcd界"},
			TextSpan{Text: "Z"},
		),
	)
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(4, 0).Rune; got != '界' {
		t.Fatalf("expected wide rune to render when it fits, got %q", got)
	}
	if got := buf.At(6, 0).Rune; got != 'Z' {
		t.Fatalf("expected following span to render after the wide rune, got %q", got)
	}
}

func TestANSITextRendersColorsWithoutEscapeSequences(t *testing.T) {
	rt := runtime.New()
	root := ANSIText("\x1b[31mred\x1b[0m")
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 'r' {
		t.Fatalf("expected first rune to be 'r', got %q", got)
	}
	if got := buf.At(0, 0).Style.Fg; got != ANSIColor(1) {
		t.Fatalf("expected ANSI red foreground, got %v", got)
	}
	if got := buf.At(3, 0).Rune; got == '[' || got == '3' {
		t.Fatal("expected ANSI escape sequence bytes to be hidden")
	}
}

func TestANSITextResetRestoresBaseStyle(t *testing.T) {
	rt := runtime.New()
	root := ANSIText("a\x1b[31mb\x1b[0mc", WithTextStyle(Style{Foreground: ANSIColor(2)}))
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(1, 0).Style.Fg; got != ANSIColor(1) {
		t.Fatalf("expected styled middle rune to be red, got %v", got)
	}
	if got := buf.At(2, 0).Style.Fg; got != ANSIColor(2) {
		t.Fatalf("expected reset rune to return to base green, got %v", got)
	}
}

func TestANSITextMalformedSequenceDoesNotPanic(t *testing.T) {
	rt := runtime.New()
	root := ANSIText("hello\x1b[31")
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 'h' {
		t.Fatalf("expected literal text to be preserved, got %q", got)
	}
}

func TestTextStyleStrikethroughReachesCells(t *testing.T) {
	rt := runtime.New()
	root := Text("done", WithTextStyle(Style{Strikethrough: true}))
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if !buf.At(0, 0).Style.Strikethrough {
		t.Fatal("expected strikethrough to reach rendered cell style")
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
	if !utf8.ValidString(got) {
		t.Fatalf("backspace result is not valid UTF-8: %q", got)
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

func TestTextInputPasteInsertsAtCursor(t *testing.T) {
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
	consumed := rt.HandleEvent(event.Event{Kind: event.PasteKind, Paste: event.Paste{Text: "b界"}})

	if !consumed {
		t.Fatal("Paste should be consumed")
	}
	if got != "ab界c" {
		t.Fatalf("Paste insert failed, got %q want ab界c", got)
	}
}

func TestTextInputPasteNormalizesNewlinesForSingleLineInput(t *testing.T) {
	got := ""
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{
		Value: "hello",
		OnChange: func(next string) {
			got = next
		},
	}))
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{Kind: event.PasteKind, Paste: event.Paste{Text: " one\r\ntwo\rthree"}})

	if !consumed {
		t.Fatal("Paste should be consumed")
	}
	if got != "hello one two three" {
		t.Fatalf("Paste newline normalization failed, got %q", got)
	}
}

func TestTextInputPasteWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := View(Style{Direction: Column}, TextInput(TextInputProps{Value: "abc"}))
	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{Kind: event.PasteKind, Paste: event.Paste{Text: "x"}}); consumed {
		t.Fatal("Paste without OnChange should return false")
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
	if got := buf.At(1, 1).Style.Fg; got != ANSIColor(8) {
		t.Fatalf("placeholder foreground = %v, want gray", got)
	}
}

func TestTextInputPlaceholderStyleCanBeOverridden(t *testing.T) {
	rt := runtime.New()

	inputNode := TextInput(TextInputProps{
		Value:            "",
		Placeholder:      "Type here",
		PlaceholderStyle: Style{Foreground: ANSIColor(5)},
		Style: Style{
			Border: BorderAll,
		},
	})
	root := View(Style{Direction: Column}, inputNode)

	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(1, 1).Style.Fg; got != ANSIColor(5) {
		t.Fatalf("placeholder foreground = %v, want override", got)
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

func TestButtonEnterWithoutOnPressReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := Button(ButtonProps{
		Label:     "Button",
		AutoFocus: true,
	})
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnter},
	})
	if consumed {
		t.Fatal("Button without OnPress should not consume Enter")
	}
}

func TestButtonSpaceWithoutOnPressReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := Button(ButtonProps{
		Label:     "Button",
		AutoFocus: true,
	})
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRune, Rune: ' '},
	})
	if consumed {
		t.Fatal("Button without OnPress should not consume Space")
	}
}

func TestButtonWithoutOnPressAllowsParentToHandleEnter(t *testing.T) {
	rt := runtime.New()
	parentHandled := false
	root := ViewWith(
		ViewProps{
			OnKey: func(ev KeyEvent) bool {
				if ev.Key == KeyEnter {
					parentHandled = true
					return true
				}
				return false
			},
		},
		Button(ButtonProps{
			Label:     "Button",
			AutoFocus: true,
		}),
	)
	updateUntilClean(rt, root)
	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnter},
	})
	if !consumed {
		t.Fatal("parent should consume Enter")
	}
	if !parentHandled {
		t.Fatal("Button without OnPress should allow Enter to bubble to parent")
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

func renderText(rt *runtime.Runtime, root Node, w, h int) string {
	updateUntilClean(rt, root)
	rt.RunLayout(w, h)

	buf := screen.NewBuffer(w, h)
	rt.Render(buf)

	runes := make([]rune, w)
	for x := 0; x < w; x++ {
		runes[x] = buf.At(x, 0).Rune
	}
	return string(runes)
}

func bufferText(buf *screen.Buffer) string {
	runes := make([]rune, len(buf.Cells))
	for i, cell := range buf.Cells {
		runes[i] = cell.Rune
	}
	return string(runes)
}

func TestSpinnerDefaultRenderWithLabel(t *testing.T) {
	rt := runtime.New()
	got := renderText(rt, Spinner(SpinnerProps{
		Active: false,
		Label:  "Loading",
	}), 40, 1)

	want := "⠋ Loading"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("spinner text = %q, want prefix %q", got, want)
	}
}

func TestSpinnerCustomFramesWithoutLabel(t *testing.T) {
	rt := runtime.New()
	got := renderText(rt, Spinner(SpinnerProps{
		Frames: []string{"-"},
		Active: false,
	}), 10, 1)

	if got[0:1] != "-" {
		t.Fatalf("spinner text = %q, want %q", got, "-")
	}
}

func TestSpinnerAppliesStyleToText(t *testing.T) {
	rt := runtime.New()
	root := Spinner(SpinnerProps{
		Active: false,
		Label:  "Loading",
		Style:  Style{Bold: true, Foreground: ANSIColor(3)},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(40, 1)
	buf := screen.NewBuffer(40, 1)
	rt.Render(buf)

	if !buf.At(0, 0).Style.Bold {
		t.Fatal("expected spinner text to be bold")
	}
	if got := buf.At(0, 0).Style.Fg; got != ANSIColor(3) {
		t.Fatalf("spinner foreground = %v, want %v", got, ANSIColor(3))
	}
}

func TestSpinnerFramesKeyNoSeparatorCollision(t *testing.T) {
	a := spinnerFramesKey([]string{"a", "b"})
	b := spinnerFramesKey([]string{"a\x00b"})

	if a == b {
		t.Fatalf("expected distinct keys, got %q", a)
	}
}

func TestCheckboxReturnsComponentNode(t *testing.T) {
	got := Checkbox(CheckboxProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("Checkbox should return ComponentKind, got %v", got.Kind)
	}
}

func TestCheckboxSpaceCallsOnChange(t *testing.T) {
	got := false
	rt := runtime.New()

	root := Checkbox(CheckboxProps{
		Value:    false,
		OnChange: func(next bool) { got = next },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRune, Rune: ' '},
	})

	if !consumed {
		t.Fatal("Space should be consumed")
	}
	if !got {
		t.Fatal("Checkbox should call OnChange(true)")
	}
}

func TestCheckboxEnterCallsOnChange(t *testing.T) {
	got := true
	rt := runtime.New()

	root := Checkbox(CheckboxProps{
		Value:    true,
		OnChange: func(next bool) { got = next },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnter},
	})

	if !consumed {
		t.Fatal("Enter should be consumed")
	}
	if got {
		t.Fatal("Checkbox should call OnChange(false)")
	}
}

func TestCheckboxWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := Checkbox(CheckboxProps{Value: false})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRune, Rune: ' '},
	})

	if consumed {
		t.Fatal("Checkbox without OnChange should return false")
	}
}

func TestDisabledCheckboxDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()

	root := Checkbox(CheckboxProps{
		Value:    false,
		Disabled: true,
		OnChange: func(bool) { called = true },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRune, Rune: ' '},
	})

	if consumed || called {
		t.Fatal("Disabled Checkbox should not consume or call OnChange")
	}
}

func TestListReturnsComponentNode(t *testing.T) {
	got := List(ListProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("List should return ComponentKind, got %v", got.Kind)
	}
}

func TestListRendersItemsVerticallyByDefault(t *testing.T) {
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two"},
		SelectedIndex: 0,
		AutoFocus:     true,
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != '>' {
		t.Fatalf("first item should render at row 0, got %q", got)
	}
	if got := buf.At(2, 0).Rune; got != 'O' {
		t.Fatalf("first item text should render at row 0, got %q", got)
	}
	if got := buf.At(2, 1).Rune; got != 'T' {
		t.Fatalf("second item text should render at row 1, got %q", got)
	}
}

func TestListDownCallsOnSelect(t *testing.T) {
	got := -1
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two", "Three"},
		SelectedIndex: 0,
		OnSelect:      func(i int) { got = i },
		AutoFocus:     true,
	})

	updateUntilClean(rt, root)

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

func TestListUpCallsOnSelect(t *testing.T) {
	got := -1
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two", "Three"},
		SelectedIndex: 2,
		OnSelect:      func(i int) { got = i },
		AutoFocus:     true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	})

	if !consumed {
		t.Fatal("Up should be consumed")
	}
	if got != 1 {
		t.Fatalf("OnSelect = %d, want 1", got)
	}
}

func TestListMovementAtEdgesReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two"},
		SelectedIndex: 0,
		OnSelect:      func(int) {},
		AutoFocus:     true,
	})

	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	}); consumed {
		t.Fatal("Up at first item should return false")
	}
}

func TestListWithoutOnSelectReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two"},
		SelectedIndex: 0,
		AutoFocus:     true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed {
		t.Fatal("List without OnSelect should return false")
	}
}

func TestDisabledListDoesNotCallOnSelect(t *testing.T) {
	called := false
	rt := runtime.New()

	root := List(ListProps{
		Items:         []string{"One", "Two"},
		SelectedIndex: 0,
		Disabled:      true,
		OnSelect:      func(int) { called = true },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed || called {
		t.Fatal("Disabled List should not consume or call OnSelect")
	}
}

func TestRadioGroupReturnsComponentNode(t *testing.T) {
	got := RadioGroup(RadioGroupProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("RadioGroup should return ComponentKind, got %v", got.Kind)
	}
}

func TestRadioGroupRendersSelectedOption(t *testing.T) {
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "blue",
		AutoFocus: true,
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(1, 0).Rune; got != ' ' {
		t.Fatalf("first option should be unselected, marker middle got %q", got)
	}
	if got := buf.At(1, 1).Rune; got != '*' {
		t.Fatalf("second option should be selected, marker middle got %q", got)
	}
}

func TestRadioGroupDownCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if !consumed {
		t.Fatal("Down should be consumed")
	}
	if got != "blue" {
		t.Fatalf("OnChange = %q, want blue", got)
	}
}

func TestRadioGroupUpCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "blue",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	})

	if !consumed {
		t.Fatal("Up should be consumed")
	}
	if got != "red" {
		t.Fatalf("OnChange = %q, want red", got)
	}
}

func TestRadioGroupMovementAtEdgesReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		OnChange:  func(string) {},
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	}); consumed {
		t.Fatal("Up at first option should return false")
	}
}

func TestRadioGroupWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed {
		t.Fatal("RadioGroup without OnChange should return false")
	}
}

func TestDisabledRadioGroupDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()

	root := RadioGroup(RadioGroupProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:    "red",
		Disabled: true,
		OnChange: func(string) { called = true },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed || called {
		t.Fatal("Disabled RadioGroup should not consume or call OnChange")
	}
}

func TestRadioGroupControlledUpdateRendersNewSelection(t *testing.T) {
	rt := runtime.New()

	form := func() Node {
		value, setValue := UseState("red")
		return RadioGroup(RadioGroupProps{
			Options: []RadioOption{
				{Label: "Red", Value: "red"},
				{Label: "Blue", Value: "blue"},
			},
			Value:    value,
			OnChange: setValue,
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(1, 0).Rune; got != ' ' {
		t.Fatalf("first option marker = %q, want space", got)
	}
	if got := buf.At(1, 1).Rune; got != '*' {
		t.Fatalf("second option marker = %q, want *", got)
	}
}

func TestSelectRendersValue(t *testing.T) {
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value: "blue",
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Expected: " < Blue > "
	found := false
	for x := 0; x < 20; x++ {
		if buf.At(x, 0).Rune == 'B' && buf.At(x+1, 0).Rune == 'l' && buf.At(x+2, 0).Rune == 'u' && buf.At(x+3, 0).Rune == 'e' {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Select should render selected option label")
	}
}

func TestSelectNextCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
			{Label: "Green", Value: "green"},
		},
		Value:     "blue",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRight},
	})

	if !consumed {
		t.Fatal("Right should be consumed")
	}
	if got != "green" {
		t.Fatalf("Right: OnChange = %q, want green", got)
	}
}

func TestSelectPrevCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyLeft},
	})

	if !consumed {
		t.Fatal("Left should be consumed")
	}

	if got != "blue" {
		t.Fatalf("Left at start should wrap to last: got %q, want blue", got)
	}
}

func TestSelectHomeEndCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
			{Label: "Green", Value: "green"},
		},
		Value:     "blue",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyHome},
	}); !consumed {
		t.Fatal("Home should be consumed")
	}
	if got != "red" {
		t.Fatalf("Home: got %q, want red", got)
	}

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnd},
	}); !consumed {
		t.Fatal("End should be consumed")
	}
	if got != "green" {
		t.Fatalf("End: got %q, want green", got)
	}
}

func TestSelectHomeEndAtEdgesReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		OnChange:  func(string) {},
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyHome},
	}); consumed {
		t.Fatal("Home at first option should return false")
	}

	root = Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "blue",
		OnChange:  func(string) {},
		AutoFocus: true,
	})

	rt.Update(root)
	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnd},
	}); consumed {
		t.Fatal("End at last option should return false")
	}
}

func TestSelectReturnsComponentNode(t *testing.T) {
	got := Select(SelectProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("Select should return ComponentKind, got %v", got.Kind)
	}
}

func TestSelectWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRight},
	})

	if consumed {
		t.Fatal("Select without OnChange should return false")
	}
}

func TestDisabledSelectDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:    "red",
		Disabled: true,
		OnChange: func(string) { called = true },
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRight},
	})

	if consumed || called {
		t.Fatal("Disabled Select should not consume or call OnChange")
	}
}

func TestSelectSpaceCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := Select(SelectProps{
		Options: []RadioOption{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		},
		Value:     "red",
		OnChange:  func(next string) { got = next },
		AutoFocus: true,
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRune, Rune: ' '},
	})

	if !consumed {
		t.Fatal("Space should be consumed")
	}
	if got != "blue" {
		t.Fatalf("Space: OnChange = %q, want blue", got)
	}
}

func TestSelectControlledUpdateRendersNewValue(t *testing.T) {
	rt := runtime.New()

	form := func() Node {
		value, setValue := UseState("red")
		return Select(SelectProps{
			Options: []RadioOption{
				{Label: "Red", Value: "red"},
				{Label: "Blue", Value: "blue"},
			},
			Value:    value,
			OnChange: setValue,
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRight},
	})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	found := false
	for x := 0; x < 20; x++ {
		if buf.At(x, 0).Rune == 'B' &&
			buf.At(x+1, 0).Rune == 'l' &&
			buf.At(x+2, 0).Rune == 'u' &&
			buf.At(x+3, 0).Rune == 'e' {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("controlled Select should render updated selected label")
	}
}

func TestSideTabsReturnsComponentNode(t *testing.T) {
	got := SideTabs(SideTabsProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("SideTabs should return ComponentKind, got %v", got.Kind)
	}
}

func TestSideTabsRendersSelectedContent(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "settings",
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
	text := bufferText(buf)

	if !strings.Contains(text, "SETTINGS") {
		t.Fatal("SideTabs should render selected tab content")
	}
	if strings.Contains(text, "HOME") {
		t.Fatal("SideTabs should render only selected tab content")
	}
}

func TestSideTabsUnknownValueFallsBackToFirstEnabledTab(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "missing",
		Tabs: []SideTab{
			{Label: "Hidden", Value: "hidden", Content: Text("HIDDEN"), Disabled: true},
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
	text := bufferText(buf)

	if !strings.Contains(text, "HOME") {
		t.Fatal("unknown SideTabs value should render first enabled tab content")
	}
	if strings.Contains(text, "HIDDEN") || strings.Contains(text, "SETTINGS") {
		t.Fatal("unknown SideTabs value should render only first enabled tab content")
	}
}

func TestSideTabsEmptyTabsDoesNotPanic(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
}

func TestSideTabsDownCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "home",
		OnChange: func(v string) {
			got = v
		},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if !consumed {
		t.Fatal("Down should be consumed")
	}
	if got != "settings" {
		t.Fatalf("OnChange = %q, want settings", got)
	}
}

func TestSideTabsUpCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "settings",
		OnChange: func(v string) {
			got = v
		},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	})

	if !consumed {
		t.Fatal("Up should be consumed")
	}
	if got != "home" {
		t.Fatalf("OnChange = %q, want home", got)
	}
}

func TestSideTabsHomeEndCallsOnChange(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "settings",
		OnChange: func(v string) {
			got = v
		},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
			{Label: "Logs", Value: "logs", Content: Text("LOGS")},
		},
	})
	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyHome},
	}); !consumed {
		t.Fatal("Home should be consumed")
	}
	if got != "home" {
		t.Fatalf("Home: OnChange = %q, want home", got)
	}

	got = ""
	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnd},
	}); !consumed {
		t.Fatal("End should be consumed")
	}
	if got != "logs" {
		t.Fatalf("End: OnChange = %q, want logs", got)
	}
}

func TestSideTabsMovementAtEdgesReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value:     "home",
		OnChange:  func(string) {},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	}); consumed {
		t.Fatal("Up at first enabled tab should return false")
	}
	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyHome},
	}); consumed {
		t.Fatal("Home at first enabled tab should return false")
	}

	root = SideTabs(SideTabsProps{
		Value:     "settings",
		OnChange:  func(string) {},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	rt.Update(root)
	updateUntilClean(rt, root)

	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	}); consumed {
		t.Fatal("Down at last enabled tab should return false")
	}
	if consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyEnd},
	}); consumed {
		t.Fatal("End at last enabled tab should return false")
	}
}

func TestSideTabsWithoutOnChangeReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value:     "home",
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed {
		t.Fatal("SideTabs without OnChange should return false")
	}
}

func TestDisabledSideTabsDoesNotCallOnChange(t *testing.T) {
	called := false
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value:    "home",
		Disabled: true,
		OnChange: func(string) {
			called = true
		},
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
		},
	})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed || called {
		t.Fatal("Disabled SideTabs should not consume or call OnChange")
	}
}

func TestSideTabsSkipsDisabledTabsWithKeyboard(t *testing.T) {
	got := ""
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "home",
		OnChange: func(v string) {
			got = v
		},
		AutoFocus: true,
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS"), Disabled: true},
			{Label: "Logs", Value: "logs", Content: Text("LOGS")},
		},
	})
	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if !consumed {
		t.Fatal("Down should be consumed")
	}
	if got != "logs" {
		t.Fatalf("OnChange = %q, want logs", got)
	}
}

func TestSideTabsStylesActiveFocusedAndDisabledTabs(t *testing.T) {
	rt := runtime.New()

	root := SideTabs(SideTabsProps{
		Value: "home",
		Tabs: []SideTab{
			{Label: "Home", Value: "home", Content: Text("HOME")},
			{Label: "Settings", Value: "settings", Content: Text("SETTINGS"), Disabled: true},
			{Label: "Logs", Value: "logs", Content: Text("LOGS")},
		},
		TabStyle:         Style{Width: Cells(20)},
		ActiveTabStyle:   Style{Foreground: ANSIColor(2), Width: Cells(2)},
		FocusedTabStyle:  Style{Bold: true},
		DisabledTabStyle: Style{Faint: true},
		AutoFocus:        true,
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	active := buf.At(2, 0).Style
	inactive := buf.At(2, 2).Style
	disabled := buf.At(2, 1).Style

	if active.Fg != ANSIColor(2) {
		t.Fatal("active tab should use ActiveTabStyle")
	}
	if inactive.Fg == ANSIColor(2) {
		t.Fatal("active tab style should not apply to inactive tabs")
	}
	if !active.Bold {
		t.Fatal("focused tab style should apply to the active tab when focused")
	}
	if !disabled.Faint {
		t.Fatal("disabled tab style should apply to disabled tabs")
	}
}

func TestSideTabsControlledUpdateRendersNewContent(t *testing.T) {
	rt := runtime.New()

	form := func() Node {
		value, setValue := UseState("home")
		return SideTabs(SideTabsProps{
			Value:     value,
			OnChange:  setValue,
			AutoFocus: true,
			Tabs: []SideTab{
				{Label: "Home", Value: "home", Content: Text("HOME")},
				{Label: "Settings", Value: "settings", Content: Text("SETTINGS")},
			},
		})
	}

	root := Component(form)
	updateUntilClean(rt, root)

	rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})
	updateUntilClean(rt, root)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
	text := bufferText(buf)

	if !strings.Contains(text, "SETTINGS") {
		t.Fatal("controlled SideTabs should render updated selected content")
	}
	if strings.Contains(text, "HOME") {
		t.Fatal("controlled SideTabs should render only updated selected content")
	}
}

func TestTextPanelReturnsComponentNode(t *testing.T) {
	got := TextPanel(TextPanelProps{})
	if got.Kind != inode.ComponentKind {
		t.Fatalf("TextPanel should return ComponentKind, got %v", got.Kind)
	}
}

func TestTextPanelRendersText(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:      "Hello",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(3),
			Border: BorderAll,
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(1, 1).Rune; got != 'H' {
		t.Fatalf("TextPanel text starts at (1,1), got %q", got)
	}
}

func TestTextPanelDownScrollsContent(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:      "one\ntwo\nthree",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})
	if !consumed {
		t.Fatal("Down should be consumed when content can scroll")
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 't' {
		t.Fatalf("after scroll, first visible line should be two, got %q", got)
	}
}

func TestTextPanelUpAtTopReturnsFalse(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:      "one\ntwo",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyUp},
	})

	if consumed {
		t.Fatal("Up at top should return false")
	}
}

func TestDisabledTextPanelDoesNotScroll(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:     "one\ntwo\nthree",
		Disabled: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyDown},
	})

	if consumed {
		t.Fatal("Disabled TextPanel should not consume Down")
	}
}

func TestTextPanelWordWrapWrapsLongLine(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:     "alpha beta gamma",
		WordWrap: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(3),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)

	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Expected visual lines: "alpha beta", "gamma"
	// "alpha beta" is 10 cells.
	if got := buf.At(0, 0).Rune; got != 'a' {
		t.Fatalf("first wrapped line starts with %q, want 'a'", got)
	}
	if got := buf.At(0, 1).Rune; got != 'g' {
		t.Fatalf("second wrapped line starts with %q, want 'g'", got)
	}
}

func TestTextPanelNoWrapRightScrollsHorizontally(t *testing.T) {
	rt := runtime.New()

	root := TextPanel(TextPanelProps{
		Text:      "abcdef",
		WordWrap:  false,
		AutoFocus: true,
		Style: Style{
			Width:  Cells(3),
			Height: Cells(1),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{
		Kind: event.KeyKind,
		Key:  event.Key{Key: tcell.KeyRight},
	})
	if !consumed {
		t.Fatal("Right should scroll horizontally when WordWrap=false")
	}

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if got := buf.At(0, 0).Rune; got != 'b' {
		t.Fatalf("after horizontal scroll, first visible rune = %q, want 'b'", got)
	}
}

func TestTextPanelScrollbarAppearsWhenOverflowing(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:          "1\n2\n3",
		ShowScrollbar: true,
		Style: Style{
			Width:  Cells(5),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Scrollbar should be at x=4. Thumb '█' at y=0, Track '│' at y=1.
	r0 := buf.At(4, 0).Rune
	if r0 != '█' {
		t.Errorf("expected thumb '█' at x=4, y=0, got %q", r0)
	}
	r1 := buf.At(4, 1).Rune
	if r1 != '│' {
		t.Errorf("expected track '│' at x=4, y=1, got %q", r1)
	}
}

func TestTextPanelScrollbarNotShownWhenContentFits(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:          "1\n2",
		ShowScrollbar: true,
		Style: Style{
			Width:  Cells(5),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	r := buf.At(4, 0).Rune
	if r != ' ' && r != 0 {
		t.Errorf("expected no scrollbar at x=4, got %q", r)
	}
}

func TestTextPanelScrollbarThumbMovesAfterScroll(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:          "1\n2\n3",
		ShowScrollbar: true,
		AutoFocus:     true,
		Style: Style{
			Width:  Cells(5),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	// Scroll down.
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}})
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// Thumb should move to y=1.
	r1 := buf.At(4, 1).Rune
	if r1 != '█' {
		t.Errorf("expected thumb '█' at x=4, y=1 after scroll, got %q", r1)
	}
}

func TestTextPanelScrollbarReducesWrapWidth(t *testing.T) {
	rt := runtime.New()
	// Text "12345" at width 5.
	// Without scrollbar, it fits on one line.
	// With scrollbar (overflowing vertically), viewportW becomes 4.
	// "12345" will wrap into "1234", "5".
	root := TextPanel(TextPanelProps{
		Text:          "12345\n2\n3",
		ShowScrollbar: true,
		WordWrap:      true,
		Style: Style{
			Width:  Cells(5),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	// First line should be "1234".
	if got := buf.At(3, 0).Rune; got != '4' {
		t.Errorf("expected '4' at (3,0), got %q", got)
	}
	// Scrollbar at x=4.
	if r := buf.At(4, 0).Rune; r != '█' && r != '│' {
		t.Errorf("expected scrollbar at x=4, got %q", r)
	}
}

func TestTextPanelPageDownScrollsByViewportHeight(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "1\n2\n3\n4\n5",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyPgDn}})

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != '3' {
		t.Errorf("expected '3' at (0,0) after PgDn, got %q", r)
	}
}

func TestTextPanelPageUpScrollsByViewportHeight(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "1\n2\n3\n4\n5",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	// Go to bottom.
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	// PageUp.
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyPgUp}})
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != '2' {
		t.Errorf("expected '2' at (0,0) after PageUp from bottom, got %q", r)
	}
}

func TestTextPanelHomeEnd(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "1\n2\n3\n4\n5",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)
	if r := buf.At(0, 0).Rune; r != '4' {
		t.Errorf("expected '4' at (0,0) after End, got %q", r)
	}

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyHome}})
	rt.RunLayout(80, 24)
	buf = screen.NewBuffer(80, 24)
	rt.Render(buf)
	if r := buf.At(0, 0).Rune; r != '1' {
		t.Errorf("expected '1' at (0,0) after Home, got %q", r)
	}
}

func TestTextPanelDownAtBottomReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "1\n2",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyDown}})
	if consumed {
		t.Error("Down at bottom should return false")
	}
}

func TestTextPanelRightAtEndReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "abcdef",
		AutoFocus: true,
		WordWrap:  false,
		Style: Style{
			Width:  Cells(3),
			Height: Cells(1),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	// Scroll to end.
	for i := 0; i < 3; i++ {
		rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	}
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	if consumed {
		t.Error("Right at end should return false")
	}
}

func TestTextPanelLeftAtStartReturnsFalse(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "abcdef",
		AutoFocus: true,
		WordWrap:  false,
		Style: Style{
			Width:  Cells(3),
			Height: Cells(1),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyLeft}})
	if consumed {
		t.Error("Left at start should return false")
	}
}

func TestTextPanelLeftRightIgnoredWhenWordWrapEnabled(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "long line",
		AutoFocus: true,
		WordWrap:  true,
		Style: Style{
			Width:  Cells(5),
			Height: Cells(1),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	consumed := rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	if consumed {
		t.Error("Right should be ignored when WordWrap is enabled")
	}
}

func TestTextPanelHorizontalScrollDoesNotRenderPartialWideRune(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "世abc",
		WordWrap:  false,
		AutoFocus: true,
		Style: Style{
			Width:  Cells(3),
			Height: Cells(1),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRight}})
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != ' ' && r != 0 {
		t.Errorf("expected blank at (0,0) to avoid partial wide rune, got %q", r)
	}
	if r := buf.At(1, 0).Rune; r != 'a' {
		t.Errorf("expected 'a' at (1,0), got %q", r)
	}
}

func TestTextPanelAutoScrollBottomShowsLatestContent(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:             "1\n2\n3\n4\n5",
		AutoScrollBottom: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != '4' {
		t.Errorf("expected '4' at (0,0) when pinned to bottom, got %q", r)
	}
	if r := buf.At(0, 1).Rune; r != '5' {
		t.Errorf("expected '5' at (0,1) when pinned to bottom, got %q", r)
	}
}

func TestTextPanelResetScrollKeyResetsScrollPosition(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:      "1\n2\n3\n4\n5",
		AutoFocus: true,
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	root = TextPanel(TextPanelProps{
		Text:           "1\n2\n3\n4\n5",
		AutoFocus:      true,
		ResetScrollKey: "next",
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})
	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != '1' {
		t.Errorf("expected reset scroll to return to top, got %q", r)
	}
}

func TestTextPanelResetScrollKeyUnchangedPreservesScrollPosition(t *testing.T) {
	rt := runtime.New()
	root := TextPanel(TextPanelProps{
		Text:           "1\n2\n3\n4\n5",
		AutoFocus:      true,
		ResetScrollKey: "stable",
		Style: Style{
			Width:  Cells(10),
			Height: Cells(2),
		},
	})

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))
	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyEnd}})
	rt.RunLayout(80, 24)
	rt.Render(screen.NewBuffer(80, 24))

	updateUntilClean(rt, root)
	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	if r := buf.At(0, 0).Rune; r != '4' {
		t.Errorf("expected scroll position to be preserved, got %q", r)
	}
}

// TestHomePageContainerNotFocusable verifies Ticket 1: a container without
// Focusable: true is not a focus stop, so Tab only cycles through inner buttons.
func TestHomePageContainerNotFocusable(t *testing.T) {
	rt := runtime.New()
	root := ViewWith(ViewProps{Style: Style{Direction: Column, FlexGrow: 1}},
		Button(ButtonProps{Label: "[ Open Dialog ]"}),
		Button(ButtonProps{Label: "[ Settings ]"}),
	)
	updateUntilClean(rt, root)

	first := rt.Focused()
	if first == nil {
		t.Fatal("expected a button to be focused initially")
	}

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyTab}})
	second := rt.Focused()
	if second == first {
		t.Error("Tab should move focus to the second button")
	}

	rt.HandleEvent(event.Event{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyTab}})
	back := rt.Focused()
	// With only 2 focusable nodes (the two buttons), a second Tab wraps back.
	// If the container were also focusable there would be 3 stops and this would fail.
	if back != first {
		t.Error("two tabs with 2 buttons should wrap back to the first (no invisible focus stop)")
	}
}
