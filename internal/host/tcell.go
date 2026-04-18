package host

import (
	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/event"
	"github.com/smason/earlgray/internal/input"
	"github.com/smason/earlgray/internal/screen"
)

// TcellHost wraps a tcell.Screen to implement Host.
type TcellHost struct {
	s tcell.Screen
}

// NewTcellHost creates a new TcellHost backed by a real terminal.
func NewTcellHost() (*TcellHost, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	return &TcellHost{s: s}, nil
}

// Init initializes the underlying tcell screen.
func (h *TcellHost) Init() error {
	if err := h.s.Init(); err != nil {
		return err
	}
	h.s.EnableMouse()
	return nil
}

// Close finalizes the tcell screen.
func (h *TcellHost) Close() error {
	h.s.Fini()
	return nil
}

// Size returns the terminal dimensions.
func (h *TcellHost) Size() (int, int) {
	return h.s.Size()
}

// PollEvent blocks for the next tcell event and converts it to an internal Event.
func (h *TcellHost) PollEvent() event.Event {
	for {
		ev := h.s.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventKey:
			return event.Event{
				Kind: event.KeyKind,
				Key: event.Key{
					Key:  e.Key(),
					Rune: e.Rune(),
					Mod:  e.Modifiers(),
				},
			}
		case *tcell.EventResize:
			w, h := e.Size()
			return event.Event{
				Kind:   event.ResizeKind,
				Width:  w,
				Height: h,
			}
		case *tcell.EventFocus:
			if e.Focused {
				return event.Event{Kind: event.FocusKind}
			}
			return event.Event{Kind: event.BlurKind}
		case *tcell.EventMouse:
			x, y := e.Position()
			return event.Event{
				Kind: event.MouseKind,
				Mouse: event.Mouse{
					X:      x,
					Y:      y,
					Button: normalizeTcellMouseButton(e.Buttons()),
					Mod:    e.Modifiers(),
				},
			}
		case nil:
			// Screen was finalized.
			return event.Event{Kind: event.QuitKind}
		}
		// Ignore other event types - loop again.
	}
}

func normalizeTcellMouseButton(b tcell.ButtonMask) input.MouseButton {
	var out input.MouseButton
	if b&tcell.Button1 != 0 {
		out |= input.MouseLeft
	}
	if b&tcell.Button2 != 0 {
		out |= input.MouseMiddle
	}
	if b&tcell.Button3 != 0 {
		out |= input.MouseRight
	}
	if b&tcell.WheelUp != 0 {
		out |= input.MouseWheelUp
	}
	if b&tcell.WheelDown != 0 {
		out |= input.MouseWheelDown
	}
	return out
}

// Show flushes pending drawing operations to the screen.
func (h *TcellHost) Show() {
	h.s.Show()
}

// Sync forces a full screen redraw.
func (h *TcellHost) Sync() {
	h.s.Sync()
}

// SetCell draws a single cell at (x, y).
func (h *TcellHost) SetCell(x, y int, ch rune, style screen.CellStyle) {
	tcellStyle := tcell.StyleDefault.
		Foreground(style.Fg.ToTcell()).
		Background(style.Bg.ToTcell())
	if style.Bold {
		tcellStyle = tcellStyle.Bold(true)
	}
	if style.Italic {
		tcellStyle = tcellStyle.Italic(true)
	}
	if style.Underline {
		tcellStyle = tcellStyle.Underline(true)
	}
	h.s.SetContent(x, y, ch, nil, tcellStyle)
}

// ShowCursor moves the cursor to (x, y).
func (h *TcellHost) ShowCursor(x, y int) {
	h.s.ShowCursor(x, y)
}

// HideCursor hides the terminal cursor.
func (h *TcellHost) HideCursor() {
	h.s.HideCursor()
}
