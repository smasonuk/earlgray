package runtime

import (
	"testing"

	"github.com/smasonuk/earlgray/internal/input"
	"github.com/smasonuk/earlgray/internal/layout"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/style"
)

// makeTextAreaInst creates a mounted textarea instance for testing.
func makeTextAreaInst(text string, onChange func(string), onCopy func(string)) (*Runtime, *Instance) {
	rt := New()
	n := &node.Node{
		Kind: node.TextAreaKind,
		Text: text,
		TextAreaOpts: node.TextAreaOptions{
			OnChange: onChange,
			OnCopy:   onCopy,
		},
		Focusable: true,
	}
	rt.Update(n)
	return rt, rt.focused
}

// sendKeyPress delivers a KeyPress directly to the focused textarea.
func sendKeyPress(rt *Runtime, press input.KeyPress) bool {
	if rt.focused == nil || rt.focused.nd == nil || rt.focused.nd.Kind != node.TextAreaKind {
		return false
	}
	result := handleTextAreaKey(rt.focused, press)
	if result {
		rt.MarkDirty()
	}
	return result
}

// sendPasteText delivers paste text to the focused textarea.
func sendPasteText(rt *Runtime, text string) bool {
	result := rt.handlePaste(text)
	if result {
		rt.MarkDirty()
	}
	return result
}

// --- Test: Selection range normalization ---

func TestTextAreaSelectionRange(t *testing.T) {
	t.Run("AnchorBeforeCursor", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 2, textAreaCursor: 7}
		start, end, ok := textAreaSelectionRange(inst, 10)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if start != 2 || end != 7 {
			t.Errorf("expected [2,7), got [%d,%d)", start, end)
		}
	})

	t.Run("AnchorAfterCursor", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 7, textAreaCursor: 2}
		start, end, ok := textAreaSelectionRange(inst, 10)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if start != 2 || end != 7 {
			t.Errorf("expected [2,7), got [%d,%d)", start, end)
		}
	})

	t.Run("EmptySelection", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 3, textAreaCursor: 3}
		_, _, ok := textAreaSelectionRange(inst, 10)
		if ok {
			t.Error("expected ok=false for empty selection")
		}
	})

	t.Run("NoAnchor", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: -1, textAreaCursor: 5}
		_, _, ok := textAreaSelectionRange(inst, 10)
		if ok {
			t.Error("expected ok=false when anchor is -1")
		}
	})

	t.Run("ClampedOutOfRange", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 100, textAreaCursor: 0}
		start, end, ok := textAreaSelectionRange(inst, 10)
		if !ok {
			t.Fatal("expected ok=true after clamping")
		}
		if start != 0 || end != 10 {
			t.Errorf("expected [0,10), got [%d,%d)", start, end)
		}
	})
}

// --- Test: Delete selection ---

func TestTextAreaDeleteSelectionRunes(t *testing.T) {
	runes := []rune("hello world")

	t.Run("DeleteMiddle", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 6, textAreaCursor: 11}
		next, cur, ok := textAreaDeleteSelectionRunes(inst, runes)
		if !ok {
			t.Fatal("expected deletion")
		}
		if string(next) != "hello " {
			t.Errorf("expected \"hello \", got %q", string(next))
		}
		if cur != 6 {
			t.Errorf("expected cursor at 6, got %d", cur)
		}
	})

	t.Run("DeleteFromStart", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: 0, textAreaCursor: 5}
		next, cur, ok := textAreaDeleteSelectionRunes(inst, runes)
		if !ok {
			t.Fatal("expected deletion")
		}
		if string(next) != " world" {
			t.Errorf("expected \" world\", got %q", string(next))
		}
		if cur != 0 {
			t.Errorf("expected cursor at 0, got %d", cur)
		}
	})

	t.Run("NoSelection", func(t *testing.T) {
		inst := &Instance{textAreaSelectionAnchor: -1, textAreaCursor: 5}
		_, _, ok := textAreaDeleteSelectionRunes(inst, runes)
		if ok {
			t.Error("expected no deletion without selection")
		}
	})
}

// --- Test: Delete selection via key ---

func TestTextAreaDeleteSelectionViaKey(t *testing.T) {
	var currentText string
	rt, inst := makeTextAreaInst("hello world", func(s string) { currentText = s }, nil)

	// Select "hello" (indices 0..5)
	inst.textAreaSelectionAnchor = 0
	inst.textAreaCursor = 5

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyBackspace})
	if !consumed {
		t.Error("expected event consumed")
	}
	if currentText != " world" {
		t.Errorf("expected \" world\", got %q", currentText)
	}
	if inst.textAreaSelectionAnchor != -1 {
		t.Errorf("expected selection cleared after delete")
	}
}

func TestTextAreaDeleteKeyDeletesSelection(t *testing.T) {
	var currentText string
	rt, inst := makeTextAreaInst("hello world", func(s string) { currentText = s }, nil)

	// Select "world" (indices 6..11)
	inst.textAreaSelectionAnchor = 6
	inst.textAreaCursor = 11

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyDelete})
	if !consumed {
		t.Error("expected event consumed")
	}
	if currentText != "hello " {
		t.Errorf("expected \"hello \", got %q", currentText)
	}
}

// --- Test: Type replaces selection ---

func TestTextAreaTypeReplacesSelection(t *testing.T) {
	var currentText string
	rt, inst := makeTextAreaInst("hello world", func(s string) { currentText = s }, nil)

	// Select "world" (indices 6..11)
	inst.textAreaSelectionAnchor = 6
	inst.textAreaCursor = 11

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyRune, Rune: 'X'})
	if !consumed {
		t.Error("expected event consumed")
	}
	if currentText != "hello X" {
		t.Errorf("expected \"hello X\", got %q", currentText)
	}
	if inst.textAreaSelectionAnchor != -1 {
		t.Errorf("expected selection cleared, anchor=%d", inst.textAreaSelectionAnchor)
	}
}

func TestTextAreaEnterReplacesSelection(t *testing.T) {
	var currentText string
	rt, inst := makeTextAreaInst("hello world", func(s string) { currentText = s }, nil)

	// Select " world" (indices 5..11)
	inst.textAreaSelectionAnchor = 5
	inst.textAreaCursor = 11

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyEnter})
	if !consumed {
		t.Error("expected event consumed")
	}
	if currentText != "hello\n" {
		t.Errorf("expected \"hello\\n\", got %q", currentText)
	}
}

// --- Test: Paste replaces selection ---

func TestTextAreaPasteReplacesSelection(t *testing.T) {
	var currentText string
	rt, inst := makeTextAreaInst("foo bar", func(s string) { currentText = s }, nil)

	// Select "foo" (indices 0..3)
	inst.textAreaSelectionAnchor = 0
	inst.textAreaCursor = 3

	consumed := sendPasteText(rt, "one\ntwo")
	if !consumed {
		t.Error("expected event consumed")
	}
	if currentText != "one\ntwo bar" {
		t.Errorf("expected \"one\\ntwo bar\", got %q", currentText)
	}
	if inst.textAreaSelectionAnchor != -1 {
		t.Errorf("expected selection cleared after paste")
	}
}

// --- Test: Copy selected text via OnCopy ---

func TestTextAreaCopyOnCopy(t *testing.T) {
	var copied string
	rt, inst := makeTextAreaInst("hello world", nil, func(s string) { copied = s })

	// Select "hello" (indices 0..5)
	inst.textAreaSelectionAnchor = 0
	inst.textAreaCursor = 5

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyCtrlC})
	if !consumed {
		t.Error("expected event consumed")
	}
	if copied != "hello" {
		t.Errorf("expected \"hello\", got %q", copied)
	}
	// Selection must be retained after copy
	if inst.textAreaSelectionAnchor != 0 {
		t.Errorf("expected selection retained after copy, anchor=%d", inst.textAreaSelectionAnchor)
	}
}

func TestTextAreaCopyNoSelectionIsNoop(t *testing.T) {
	var copied string
	rt, inst := makeTextAreaInst("hello world", nil, func(s string) { copied = s })

	inst.textAreaSelectionAnchor = -1
	inst.textAreaCursor = 3

	consumed := sendKeyPress(rt, input.KeyPress{Key: input.KeyCtrlC})
	if consumed {
		t.Error("expected event not consumed (no selection)")
	}
	if copied != "" {
		t.Errorf("expected OnCopy not called, got %q", copied)
	}
}

// --- Test: Shift+Arrow creates/extends selection ---

func TestTextAreaShiftArrowSelection(t *testing.T) {
	t.Run("ShiftRightCreatesSelection", func(t *testing.T) {
		rt, inst := makeTextAreaInst("hello", nil, nil)
		inst.textAreaCursor = 0
		inst.textAreaSelectionAnchor = -1

		sendKeyPress(rt, input.KeyPress{Key: input.KeyRight, Mod: input.ModShift})

		if inst.textAreaSelectionAnchor != 0 {
			t.Errorf("expected anchor=0, got %d", inst.textAreaSelectionAnchor)
		}
		if inst.textAreaCursor != 1 {
			t.Errorf("expected cursor=1, got %d", inst.textAreaCursor)
		}
	})

	t.Run("ShiftLeftCreatesSelection", func(t *testing.T) {
		rt, inst := makeTextAreaInst("hello", nil, nil)
		inst.textAreaCursor = 3
		inst.textAreaSelectionAnchor = -1

		sendKeyPress(rt, input.KeyPress{Key: input.KeyLeft, Mod: input.ModShift})

		if inst.textAreaSelectionAnchor != 3 {
			t.Errorf("expected anchor=3, got %d", inst.textAreaSelectionAnchor)
		}
		if inst.textAreaCursor != 2 {
			t.Errorf("expected cursor=2, got %d", inst.textAreaCursor)
		}
	})

	t.Run("PlainArrowClearsSelection", func(t *testing.T) {
		rt, inst := makeTextAreaInst("hello", nil, nil)
		inst.textAreaCursor = 2
		inst.textAreaSelectionAnchor = 0

		sendKeyPress(rt, input.KeyPress{Key: input.KeyRight})

		if inst.textAreaSelectionAnchor != -1 {
			t.Errorf("expected selection cleared, anchor=%d", inst.textAreaSelectionAnchor)
		}
	})

	t.Run("ShiftRightExtendsExistingSelection", func(t *testing.T) {
		rt, inst := makeTextAreaInst("hello", nil, nil)
		inst.textAreaCursor = 1
		inst.textAreaSelectionAnchor = 0

		sendKeyPress(rt, input.KeyPress{Key: input.KeyRight, Mod: input.ModShift})

		if inst.textAreaSelectionAnchor != 0 {
			t.Errorf("expected anchor stays at 0, got %d", inst.textAreaSelectionAnchor)
		}
		if inst.textAreaCursor != 2 {
			t.Errorf("expected cursor=2, got %d", inst.textAreaCursor)
		}
	})
}

// --- Test: Wheel scroll does not snap cursor back ---

func TestTextAreaWheelScrollIndependentFromCursor(t *testing.T) {
	rt := New()
	text := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	n := &node.Node{
		Kind: node.TextAreaKind,
		Text: text,
		TextAreaOpts: node.TextAreaOptions{
			OnChange: func(s string) {},
		},
		Focusable: true,
	}
	rt.Update(n)
	inst := rt.focused

	// Set content area small enough that lines overflow.
	inst.layout = layout.Result{
		Content: style.Rect{X: 0, Y: 0, W: 20, H: 3},
	}

	// Move cursor to start and set flag.
	inst.textAreaCursor = 0
	inst.textAreaEnsureCursorVisible = true

	// Wheel scroll down (delta=+1) should move scrollY and clear EnsureCursorVisible.
	scrolled := scrollTextArea(inst, 1)
	if !scrolled {
		t.Error("expected scrollTextArea to return true with 10 lines and H=3")
	}

	if inst.textAreaEnsureCursorVisible {
		t.Error("expected textAreaEnsureCursorVisible=false after wheel scroll")
	}
}

// --- Test: resetTextAreaState initializes selection fields ---

func TestResetTextAreaStateInitializesSelection(t *testing.T) {
	rt := New()
	n := &node.Node{
		Kind:      node.TextAreaKind,
		Text:      "hello",
		Focusable: true,
	}
	rt.Update(n)
	inst := rt.focused

	if inst.textAreaSelectionAnchor != -1 {
		t.Errorf("expected textAreaSelectionAnchor=-1 after mount, got %d", inst.textAreaSelectionAnchor)
	}
	if inst.textAreaDragging {
		t.Error("expected textAreaDragging=false after mount")
	}
}
