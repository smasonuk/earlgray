package runtime

import (
	"testing"

	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/node"
)

func TestTextAreaPaste(t *testing.T) {
	t.Run("MultilinePasteAtEnd", func(t *testing.T) {
		var currentText string
		rt := New()
		n := &node.Node{
			Kind: node.TextAreaKind,
			Text: "hello ",
			TextAreaOpts: node.TextAreaOptions{
				OnChange: func(s string) { currentText = s },
			},
			Focusable: true,
		}
		rt.Update(n)
		// Ensure focused is set
		if rt.focused == nil {
			t.Fatal("expected textarea to be focused")
		}
		rt.focused.textAreaCursor = 6

		consumed := rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "one\ntwo\nthree"},
		})

		if !consumed {
			t.Errorf("expected event to be consumed")
		}

		expected := "hello one\ntwo\nthree"
		if currentText != expected {
			t.Errorf("Expected %q, got %q", expected, currentText)
		}
		if rt.focused.textAreaCursor != len([]rune(expected)) {
			t.Errorf("Expected cursor at %d, got %d", len([]rune(expected)), rt.focused.textAreaCursor)
		}
	})

	t.Run("PasteAtMiddle", func(t *testing.T) {
		var currentText string
		rt := New()
		n := &node.Node{
			Kind: node.TextAreaKind,
			Text: "hello world",
			TextAreaOpts: node.TextAreaOptions{
				OnChange: func(s string) { currentText = s },
			},
			Focusable: true,
		}
		rt.Update(n)
		rt.focused.textAreaCursor = 6 // after "hello "

		rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "beautiful "},
		})

		expected := "hello beautiful world"
		if currentText != expected {
			t.Errorf("Expected %q, got %q", expected, currentText)
		}
	})

	t.Run("LineEndingNormalization", func(t *testing.T) {
		var currentText string
		rt := New()
		n := &node.Node{
			Kind: node.TextAreaKind,
			Text: "",
			TextAreaOpts: node.TextAreaOptions{
				OnChange: func(s string) { currentText = s },
			},
			Focusable: true,
		}
		rt.Update(n)

		rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "a\r\nb\rc"},
		})

		expected := "a\nb\nc"
		if currentText != expected {
			t.Errorf("Expected %q, got %q", expected, currentText)
		}
	})

	t.Run("TabsPreserved", func(t *testing.T) {
		var currentText string
		rt := New()
		n := &node.Node{
			Kind: node.TextAreaKind,
			Text: "",
			TextAreaOpts: node.TextAreaOptions{
				OnChange: func(s string) { currentText = s },
			},
			Focusable: true,
		}
		rt.Update(n)

		rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "if true {\n\tprintln()\n}"},
		})

		expected := "if true {\n\tprintln()\n}"
		if currentText != expected {
			t.Errorf("Expected %q, got %q", expected, currentText)
		}
	})

	t.Run("DisabledIgnored", func(t *testing.T) {
		var currentText string
		rt := New()
		n := &node.Node{
			Kind:     node.TextAreaKind,
			Text:     "initial",
			Disabled: true,
			TextAreaOpts: node.TextAreaOptions{
				OnChange: func(s string) { currentText = s },
			},
			Focusable: true,
		}
		rt.Update(n)

		consumed := rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "pasted"},
		})

		if consumed {
			t.Error("Expected event not to be consumed for disabled textarea")
		}
		if currentText != "" {
			t.Errorf("Expected no change, got %q", currentText)
		}
	})

	t.Run("NilOnChangeIgnored", func(t *testing.T) {
		rt := New()
		n := &node.Node{
			Kind: node.TextAreaKind,
			Text: "initial",
			TextAreaOpts: node.TextAreaOptions{
				OnChange: nil,
			},
			Focusable: true,
		}
		rt.Update(n)

		consumed := rt.HandleEvent(event.Event{
			Kind:  event.PasteKind,
			Paste: event.Paste{Text: "pasted"},
		})

		if consumed {
			t.Error("Expected event not to be consumed for nil OnChange")
		}
	})
}
