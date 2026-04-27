package tui

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/smasonuk/earlgray/internal/event"
	"github.com/smasonuk/earlgray/internal/host"
	"github.com/smasonuk/earlgray/internal/screen"
)

// fakeHost is a test double for host.Host that records rendered cells and
// cursor state. PollEvent pops from a queue; when empty it returns QuitKind.
type fakeHost struct {
	w, h int

	events []event.Event

	cells         []rune // w*h grid tracking last written rune per cell
	cursorVisible bool
	cursorX       int
	cursorY       int
	showCount     int
	initCalled    bool
	closeCalled   bool
	onShow        func(*fakeHost)
}

func newFakeHost(w, h int, events []event.Event) *fakeHost {
	cells := make([]rune, w*h)
	for i := range cells {
		cells[i] = ' '
	}
	return &fakeHost{w: w, h: h, events: events, cells: cells}
}

func (f *fakeHost) Init() error {
	f.initCalled = true
	return nil
}

func (f *fakeHost) Close() error {
	f.closeCalled = true
	return nil
}

func (f *fakeHost) Size() (int, int) { return f.w, f.h }

func (f *fakeHost) PollEvent() event.Event {
	if len(f.events) == 0 {
		return event.Event{Kind: event.QuitKind}
	}
	ev := f.events[0]
	f.events = f.events[1:]
	return ev
}

func (f *fakeHost) Show() {
	f.showCount++
	if f.onShow != nil {
		f.onShow(f)
	}
}
func (f *fakeHost) Sync() {}
func (f *fakeHost) ShowCursor(x, y int) {
	f.cursorVisible = true
	f.cursorX = x
	f.cursorY = y
}
func (f *fakeHost) HideCursor() { f.cursorVisible = false }

func (f *fakeHost) SetCell(x, y int, ch rune, _ screen.CellStyle) {
	if x >= 0 && x < f.w && y >= 0 && y < f.h {
		f.cells[y*f.w+x] = ch
	}
}

func (f *fakeHost) screenText() string {
	return string(f.cells)
}

func (f *fakeHost) containsText(s string) bool {
	return strings.Contains(f.screenText(), s)
}

// Verify fakeHost satisfies the host.Host interface at compile time.
var _ host.Host = (*fakeHost)(nil)

type blockingFakeHost struct {
	*fakeHost
	events    chan event.Event
	closeCh   chan struct{}
	closeOnce sync.Once
}

func newBlockingFakeHost(w, h int) *blockingFakeHost {
	return &blockingFakeHost{
		fakeHost: newFakeHost(w, h, nil),
		events:   make(chan event.Event, 16),
		closeCh:  make(chan struct{}),
	}
}

func (f *blockingFakeHost) PollEvent() event.Event {
	select {
	case ev := <-f.events:
		return ev
	case <-f.closeCh:
		return event.Event{Kind: event.QuitKind}
	}
}

func (f *blockingFakeHost) Close() error {
	f.closeOnce.Do(func() {
		f.fakeHost.closeCalled = true
		close(f.closeCh)
	})
	return nil
}

// TestRunInitialRenderDrainsFocusDirtyState verifies that after the initial
// render, the frame shows the focused style — not the unfocused style that
// would appear if only one render pass ran before ensureFocus set dirty.
func TestRunInitialRenderDrainsFocusDirtyState(t *testing.T) {
	app := func() Node {
		return Component(func() Node {
			focused := UseFocused()
			label := "blurred"
			if focused {
				label = "focused"
			}
			return ViewWith(
				ViewProps{Focusable: true},
				Text(label),
			)
		})
	}

	fake := newFakeHost(80, 24, nil) // no events → immediate quit
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("focused") {
		t.Errorf("expected final frame to contain %q; screen: %q", "focused", fake.screenText()[:40])
	}
}

// TestRunPostEventRenderDrainsDirtyState verifies that after a key event causes
// a focus-scope change, Run immediately renders again to settle the dirty state
// rather than waiting for another event.
func TestRunPostEventRenderDrainsDirtyState(t *testing.T) {
	app := func() Node {
		return Component(func() Node {
			open, setOpen := UseState(false)

			page := ViewWith(
				ViewProps{
					Focusable: true,
					OnKey: func(ev KeyEvent) bool {
						if ev.Key == KeyRune && ev.Rune == 'o' {
							setOpen(true)
							return true
						}
						return false
					},
				},
				Text("page"),
			)

			if !open {
				return page
			}

			return Overlay(
				page,
				Dialog(DialogProps{}, ViewWith(
					ViewProps{Focusable: true, AutoFocus: true},
					Text("dialog"),
				)),
			)
		})
	}

	events := []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'o'}},
		// QuitKind will be returned automatically when the queue is empty.
	}
	fake := newFakeHost(80, 24, events)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("dialog") {
		t.Errorf("expected final frame to contain %q; screen: %q", "dialog", fake.screenText()[:40])
	}
}

// TestRunCursorVisibleForFocusedTextInput verifies that after the initial
// render, ShowCursor is called when a TextInput with AutoFocus is the root.
func TestRunCursorVisibleForFocusedTextInput(t *testing.T) {
	app := func() Node {
		return TextInput(TextInputProps{
			Value:     "abc",
			AutoFocus: true,
		})
	}

	fake := newFakeHost(80, 24, nil)
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if !fake.cursorVisible {
		t.Error("expected cursor to be visible for focused TextInput")
	}
	if fake.cursorX < 0 || fake.cursorY < 0 {
		t.Errorf("cursor position (%d,%d) should be non-negative", fake.cursorX, fake.cursorY)
	}
}

func TestRunWithOptionsPostTriggersRerender(t *testing.T) {
	value := "before"
	app := func() Node {
		return Text(value)
	}

	fake := newBlockingFakeHost(80, 24)
	err := runWithHost(app, RunOptions{
		OnStart: func(h AppHandle) {
			h.Post(func() {
				value = "after"
				h.Quit()
			})
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("after") {
		t.Fatalf("expected rerendered frame to contain %q; screen: %q", "after", fake.screenText()[:40])
	}
}

func TestRunUseEffectRunsAfterShow(t *testing.T) {
	fake := newFakeHost(80, 24, nil)
	showCountSeen := 0

	app := func() Node {
		return Component(func() Node {
			UseEffect(func() func() {
				showCountSeen = fake.showCount
				return nil
			})
			return Text("hello")
		})
	}

	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if showCountSeen == 0 {
		t.Fatalf("effect should run after Show; observed showCount=%d", showCountSeen)
	}
}

func TestRunUseEffectAsyncStateSetterRerenders(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)

	app := func() Node {
		return Component(func() Node {
			label, setLabel := UseState("before")
			UseEffect(func() func() {
				go func() {
					time.Sleep(5 * time.Millisecond)
					setLabel("after")
				}()
				return nil
			})
			return Text(label)
		})
	}

	err := runWithHost(app, RunOptions{
		OnStart: func(h AppHandle) {
			go func() {
				time.Sleep(30 * time.Millisecond)
				h.Quit()
			}()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("after") {
		t.Fatalf("expected async setter rerender to contain %q; screen: %q", "after", fake.screenText()[:40])
	}
}

func TestSpinnerActiveAdvancesFrames(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)
	frames := make([]string, 100)
	for i := range frames {
		frames[i] = string(rune('0' + (i % 10)))
	}

	err := runWithHost(func() Node {
		return Spinner(SpinnerProps{
			Frames:   frames,
			Active:   true,
			Interval: 5 * time.Millisecond,
		})
	}, RunOptions{
		OnStart: func(h AppHandle) {
			go func() {
				time.Sleep(20 * time.Millisecond)
				h.Quit()
			}()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if fake.cells[0] == '0' {
		t.Fatalf("expected spinner frame to advance from initial frame; screen: %q", fake.screenText()[:40])
	}
}

func TestSpinnerInactiveDoesNotAdvance(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)
	frames := make([]string, 100)
	for i := range frames {
		frames[i] = string(rune('0' + (i % 10)))
	}

	err := runWithHost(func() Node {
		return Spinner(SpinnerProps{
			Frames:   frames,
			Active:   false,
			Interval: 5 * time.Millisecond,
		})
	}, RunOptions{
		OnStart: func(h AppHandle) {
			go func() {
				time.Sleep(20 * time.Millisecond)
				h.Quit()
			}()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if fake.cells[0] != '0' {
		t.Fatalf("expected inactive spinner to keep initial frame; screen: %q", fake.screenText()[:40])
	}
}

func TestSpinnerStopsPostingAfterUnmount(t *testing.T) {
	type snapshot struct {
		phase int32
		ch    rune
	}

	fake := newBlockingFakeHost(80, 24)
	var mounted atomic.Bool
	var phase atomic.Int32
	mounted.Store(true)

	var snapshots []snapshot
	fake.onShow = func(f *fakeHost) {
		snapshots = append(snapshots, snapshot{
			phase: phase.Load(),
			ch:    f.cells[0],
		})
	}

	err := runWithHost(func() Node {
		if mounted.Load() {
			return Spinner(SpinnerProps{
				Frames:   []string{"0", "1"},
				Active:   true,
				Interval: 5 * time.Millisecond,
			})
		}
		return Text("done")
	}, RunOptions{
		OnStart: func(h AppHandle) {
			go func() {
				time.Sleep(20 * time.Millisecond)
				h.Post(func() {
					mounted.Store(false)
					phase.Store(1)
				})
				time.Sleep(20 * time.Millisecond)
				h.Quit()
			}()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	var unmounted []snapshot
	for _, snap := range snapshots {
		if snap.phase == 1 {
			unmounted = append(unmounted, snap)
		}
	}

	if len(unmounted) != 1 {
		t.Fatalf("expected exactly one unmounted render, got %d (%v)", len(unmounted), unmounted)
	}
	if unmounted[0].ch != 'd' {
		t.Fatalf("expected unmounted render to show replacement content, got %q", unmounted[0].ch)
	}
}

func TestRunWithOptionsPostOrderIsPreserved(t *testing.T) {
	value := ""
	app := func() Node {
		return Text(value)
	}

	fake := newBlockingFakeHost(80, 24)
	err := runWithHost(app, RunOptions{
		OnStart: func(h AppHandle) {
			h.Post(func() { value += "a" })
			h.Post(func() {
				value += "b"
				h.Quit()
			})
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("ab") {
		t.Fatalf("expected final frame to contain %q; screen: %q", "ab", fake.screenText()[:40])
	}
}

func TestRunWithOptionsQuitExits(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)
	err := runWithHost(func() Node { return Text("hello") }, RunOptions{
		OnStart: func(h AppHandle) {
			h.Quit()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithOptionsPostAfterQuitIsSafe(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)
	err := runWithHost(func() Node { return Text("hello") }, RunOptions{
		OnStart: func(h AppHandle) {
			h.Quit()
			h.Quit()
			h.Post(func() {
				t.Fatal("posted function should not run after quit")
			})
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithOptionsCtrlCDisabledDeliversKey(t *testing.T) {
	receivedCtrlC := false

	app := func() Node {
		return ViewWith(ViewProps{
			Focusable: true,
			OnKey: func(ev KeyEvent) bool {
				if ev.Key == KeyCtrlC {
					receivedCtrlC = true
					return true
				}
				return false
			},
		})
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyCtrlC}},
	})
	err := runWithHost(app, RunOptions{
		DisableCtrlCQuit: true,
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if !receivedCtrlC {
		t.Fatal("expected Ctrl-C to be delivered to the app when built-in quit is disabled")
	}
}

func TestRunWithOptionsCtrlCQuitsByDefault(t *testing.T) {
	handled := false
	app := func() Node {
		return ViewWith(ViewProps{
			Focusable: true,
			OnKey: func(KeyEvent) bool {
				handled = true
				return true
			},
		})
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyCtrlC}},
	})
	err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if handled {
		t.Fatal("expected built-in Ctrl-C quit to stop before app key handlers run")
	}
}

func TestRunWithOptionsEveryTriggersRerender(t *testing.T) {
	count := 0
	app := func() Node {
		return Text(string(rune('0' + count)))
	}

	fake := newBlockingFakeHost(80, 24)
	err := runWithHost(app, RunOptions{
		OnStart: func(h AppHandle) {
			h.Every(5*time.Millisecond, func() {
				count++
				if count >= 3 {
					h.Quit()
				}
			})
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if count < 3 {
		t.Fatalf("expected interval to fire at least 3 times, got %d", count)
	}
	if !fake.containsText("3") {
		t.Fatalf("expected final frame to contain %q; screen: %q", "3", fake.screenText()[:40])
	}
}

func TestRunWithOptionsEveryStopStopsFutureTicks(t *testing.T) {
	count := 0
	fake := newBlockingFakeHost(80, 24)

	err := runWithHost(func() Node { return Text(string(rune('0' + count))) }, RunOptions{
		OnStart: func(h AppHandle) {
			var stop func()
			stop = h.Every(5*time.Millisecond, func() {
				count++
				if count == 2 {
					stop()
					stop()
				}
			})
			go func() {
				time.Sleep(30 * time.Millisecond)
				h.Quit()
			}()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf("expected interval to stop at 2 ticks, got %d", count)
	}
}

func TestRunWithOptionsQuitStopsIntervalWithoutPanic(t *testing.T) {
	fake := newBlockingFakeHost(80, 24)
	fired := make(chan struct{}, 1)

	err := runWithHost(func() Node { return Text("hello") }, RunOptions{
		OnStart: func(h AppHandle) {
			h.Every(50*time.Millisecond, func() {
				select {
				case fired <- struct{}{}:
				default:
				}
			})
			h.Quit()
		},
	}, func() (host.Host, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond)
	select {
	case <-fired:
		t.Fatal("interval should not fire after quit")
	default:
	}
}

func TestConsumedKeyRerendersExternalState(t *testing.T) {
	label := "before"
	app := func() Node {
		return ViewWith(ViewProps{
			Focusable: true,
			AutoFocus: true,
			OnKey: func(ev KeyEvent) bool {
				if ev.Key == KeyRune && ev.Rune == 'x' {
					label = "after"
					return true
				}
				return false
			},
		}, Text(label))
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}},
	})
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("after") {
		t.Fatalf("expected final frame to contain %q; screen: %q", "after", fake.screenText()[:40])
	}
}

func TestUnconsumedKeyDoesNotForceRerender(t *testing.T) {
	renders := 0
	app := func() Node {
		renders++
		return ViewWith(ViewProps{
			Focusable: true,
			AutoFocus: true,
			OnKey: func(ev KeyEvent) bool {
				return false
			},
		}, Text("steady"))
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyRune, Rune: 'x'}},
	})
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if renders != 2 {
		t.Fatalf("expected only initial focus-settling renders, got %d", renders)
	}
}

func TestConsumedMouseRerendersExternalState(t *testing.T) {
	label := "before"
	app := func() Node {
		return ViewWith(ViewProps{
			Focusable: true,
			AutoFocus: true,
			OnMouse: func(ev MouseEvent) bool {
				if ev.Button&MouseLeft != 0 && ev.Action == MousePress {
					label = "after"
					return true
				}
				return false
			},
		}, Text(label))
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.MouseKind, Mouse: event.Mouse{X: 0, Y: 0, Button: MouseLeft}},
	})
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if !fake.containsText("after") {
		t.Fatalf("expected final frame to contain %q; screen: %q", "after", fake.screenText()[:40])
	}
}

func TestRunWithOptionsZeroValueCtrlCQuits(t *testing.T) {
	handled := false
	app := func() Node {
		return ViewWith(ViewProps{
			Focusable: true,
			OnKey: func(KeyEvent) bool {
				handled = true
				return true
			},
		})
	}

	fake := newFakeHost(80, 24, []event.Event{
		{Kind: event.KeyKind, Key: event.Key{Key: tcell.KeyCtrlC}},
	})
	if err := runWithHost(app, RunOptions{}, func() (host.Host, error) { return fake, nil }); err != nil {
		t.Fatal(err)
	}

	if handled {
		t.Fatal("expected zero-value RunOptions to preserve default Ctrl-C quit behavior")
	}
}
