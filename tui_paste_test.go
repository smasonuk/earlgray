package tui

import (
	"testing"

	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/host"
	"github.com/smason/earlgray/internal/screen"
)

type pasteTestHost struct {
	events chan event.Event
}

func (h *pasteTestHost) Init() error { return nil }

func (h *pasteTestHost) Close() error { return nil }

func (h *pasteTestHost) Size() (int, int) { return 80, 24 }

func (h *pasteTestHost) PollEvent() event.Event {
	return <-h.events
}

func (h *pasteTestHost) Show() {}

func (h *pasteTestHost) Sync() {}

func (h *pasteTestHost) SetCell(x, y int, ch rune, style screen.CellStyle) {}

func (h *pasteTestHost) ShowCursor(x, y int) {}

func (h *pasteTestHost) HideCursor() {}

func TestRunWithHostForwardsPasteToFocusedTextArea(t *testing.T) {
	events := make(chan event.Event, 2)
	events <- event.Event{
		Kind: event.PasteKind,
		Paste: event.Paste{
			Text: "one\ntwo\tthree",
		},
	}
	events <- event.Event{Kind: event.QuitKind}

	testHost := &pasteTestHost{events: events}

	var observed string

	err := runWithHost(
		func() Node {
			return Component(func() Node {
				value, setValue := UseState("")

				return TextArea(TextAreaProps{
					Value:     value,
					AutoFocus: true,
					OnChange: func(next string) {
						observed = next
						setValue(next)
					},
				})
			})
		},
		RunOptions{},
		func() (host.Host, error) {
			return testHost, nil
		},
	)

	if err != nil {
		t.Fatalf("runWithHost returned error: %v", err)
	}

	want := "one\ntwo\tthree"
	if observed != want {
		t.Fatalf("pasted text mismatch:\nwant %q\ngot  %q", want, observed)
	}
}
