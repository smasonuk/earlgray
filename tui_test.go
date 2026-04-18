package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/runtime"
	"github.com/smason/earlgray/internal/screen"
	inode "github.com/smason/earlgray/internal/node"
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

	// After Update, ensureFocus may mark dirty if focus was set.
	// Re-update to ensure component renders with IsFocused() == true.
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
	inputNode := TextInput(props)

	// Create another focusable widget to steal focus.
	wrapperNode := View(
		Style{},
		inputNode,
		Button(ButtonProps{Label: "Button"}),
	)
	rt.Update(wrapperNode)

	rt.RunLayout(80, 24)
	buf := screen.NewBuffer(80, 24)
	rt.Render(buf)

	_, _, visible := rt.Cursor()
	if visible {
		t.Error("unfocused TextInput should not request cursor")
	}
}
