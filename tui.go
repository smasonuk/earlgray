package tui

import (
	"reflect"

	"github.com/smason/earlgray/internal/event"
	inode "github.com/smason/earlgray/internal/node"
	"github.com/smason/earlgray/internal/render"
	"github.com/smason/earlgray/internal/runtime"
	"github.com/smason/earlgray/internal/screen"

	"github.com/smason/earlgray/internal/host"
)

// Node is the opaque tree element returned by View, Text, Keyed, etc.
type Node = *inode.Node

// TextOption configures a Text node.
type TextOption func(*inode.TextOptions)

// WithAlign sets text alignment.
func WithAlign(a Align) TextOption {
	return func(opts *inode.TextOptions) {
		switch a {
		case AlignStart:
			opts.Align = inode.TextAlignLeft
		case AlignCenter:
			opts.Align = inode.TextAlignCenter
		case AlignEnd:
			opts.Align = inode.TextAlignRight
		}
	}
}

// View creates a container node with the given style and children.
func View(s Style, children ...Node) Node {
	return &inode.Node{
		Kind:     inode.ViewKind,
		Style:    s,
		Children: children,
	}
}

// Text creates a text leaf node.
func Text(value string, opts ...TextOption) Node {
	var textOpts inode.TextOptions
	for _, o := range opts {
		o(&textOpts)
	}
	return &inode.Node{
		Kind:     inode.TextKind,
		Text:     value,
		TextOpts: textOpts,
	}
}

// Keyed wraps a child node with an explicit reconciliation key.
func Keyed(key string, child Node) Node {
	return &inode.Node{
		Kind:     inode.KeyedKind,
		Key:      key,
		Children: []*inode.Node{child},
	}
}

// Component wraps a function component so it participates in the runtime.
// The function is called on every render; state is preserved across calls
// via UseState.
func Component(fn func() Node) Node {
	id := reflect.ValueOf(fn).Pointer()
	return &inode.Node{
		Kind:   inode.ComponentKind,
		CompFn: fn,
		CompID: id,
	}
}

// UseState returns the current value of a state slot and a setter function.
// It must only be called from within a component function.
func UseState[T any](initial T) (T, func(T)) {
	return runtime.UseState(initial)
}

// Run initializes the terminal, runs the main loop, and cleans up on exit.
// The root function is called on every render to produce the new node tree.
func Run(root func() Node) error {
	h, err := host.NewTcellHost()
	if err != nil {
		return err
	}
	if err := h.Init(); err != nil {
		return err
	}
	defer h.Close()

	rt := runtime.New()
	w, h2 := h.Size()

	// Initial render.
	rootNode := root()
	rt.Update(rootNode)
	rt.RunLayout(w, h2)
	buf := screen.NewBuffer(w, h2)
	rt.Render(buf)
	render.FlushDiff(nil, buf, h)
	h.Show()
	h.HideCursor()

	prev := buf

	for {
		ev := h.PollEvent()
		switch ev.Kind {
		case event.QuitKind:
			return nil
		case event.ResizeKind:
			w, h2 = ev.Width, ev.Height
			rt.MarkDirty()
		case event.KeyKind:
			// Default quit on 'q' or Ctrl-C.
			if (ev.Key.Rune == 'q' && ev.Key.Mod == 0) ||
				ev.Key.Key == 3 { // Ctrl-C
				return nil
			}
			if !rt.HandleEvent(ev) {
				// Event not consumed; re-render in case a component reacted.
				rt.MarkDirty()
			}
		}

		if rt.IsDirty() {
			rootNode = root()
			rt.Update(rootNode)
			rt.RunLayout(w, h2)
			next := screen.NewBuffer(w, h2)
			rt.Render(next)
			render.FlushDiff(prev, next, h)
			h.Show()
			prev = next
		}
	}
}
