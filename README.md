# EarlGray

A React-like retained-mode TUI library for Go, backed by [tcell](https://github.com/gdamore/tcell).

## Install

```sh
go get github.com/smason/earlgray
```

## Minimal app

```go
package main

import (
    "fmt"
    "os"

    tui "github.com/smason/earlgray"
)

func App() tui.Node {
    return tui.View(
        tui.Style{Padding: tui.All(1)},
        tui.Text("Hello EarlGray"),
    )
}

func main() {
    if err := tui.Run(App); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## Layout basics

Views are the primary container. Layout follows a subset of flexbox.

| Property | Description |
|----------|-------------|
| `Direction` | `tui.Row` (default) or `tui.Column` |
| `FlexGrow` | Integer weight for distributing remaining space |
| `Width` / `Height` | Fixed size via `tui.Cells(n)` or auto via `tui.Auto()` |
| `Padding` | Inner spacing via `tui.All(n)` or `tui.Insets{Top,Right,Bottom,Left}` |
| `Gap` | Space between children (integer cell count) |
| `Border` | `tui.BorderAll`, `tui.BorderTop`, etc. |

```go
tui.View(
    tui.Style{
        Direction: tui.Column,
        Padding:   tui.All(1),
        Gap:       1,
        FlexGrow:  1,
    },
    tui.Text("First line"),
    tui.Text("Second line"),
)
```

## Components and state

Wrap a function in `tui.Component` to give it a lifecycle. Call `tui.UseState`
inside the function to store state across renders.

`UseState` and `UseFocused` must only be called inside functions rendered
through `tui.Component`. Do not call them from plain helper functions.

```go
func Counter() tui.Node {
    count, setCount := tui.UseState(0)
    return tui.ViewWith(
        tui.ViewProps{
            Style:     tui.Style{Padding: tui.All(1)},
            Focusable: true,
            OnKey: func(ev tui.KeyEvent) bool {
                if ev.Key == tui.KeyEnter {
                    setCount(count + 1)
                    return true
                }
                return false
            },
        },
        tui.Text(fmt.Sprintf("Count: %d (Enter to increment)", count)),
    )
}

func App() tui.Node {
    return tui.Component(Counter)
}
```

If you render components in a dynamic or reordered list, wrap each in
`tui.Keyed` so reconciliation preserves the intended identity.

## Focus and keys

`ViewWith` creates a view with optional key handling and focus.

- `Focusable: true` — node participates in Tab focus traversal.
- `AutoFocus: true` — node receives focus on initial mount if nothing else is focused.
- `Disabled: true` — node is skipped in Tab traversal and does not receive key events.
- `OnKey` — called when the node (or a child) is focused and a key is pressed.
  Return `true` to consume the event; `false` to let it bubble.

Tab moves focus forward; Shift+Tab moves focus backward.

Ctrl-C always exits, handled by `tui.Run`.

## Button

`tui.Button` is a focusable button that responds to Enter and Space.

```go
tui.Button(tui.ButtonProps{
    Label:   "[ Click me ]",
    OnPress: func() { /* handle press */ },
    AutoFocus: true,
    Disabled: false,
    Style: tui.Style{
        Width:  tui.Cells(14),
        Height: tui.Cells(3),
        Border: tui.BorderAll,
    },
    FocusedStyle: tui.Style{
        Foreground: tui.ANSIColor(3), // yellow when focused
    },
})
```

- `AutoFocus: true` — button receives focus on initial mount if nothing else is focused.
- `Disabled: true` — button cannot be focused or pressed.

## TextInput

`tui.TextInput` is a controlled, single-line text entry widget. The parent owns
the value and updates it via `OnChange`.

```go
name, setName := tui.UseState("")

tui.TextInput(tui.TextInputProps{
    Value:       name,
    OnChange:    setName,
    Placeholder: "Type your name...",
    AutoFocus:   true,
    Disabled:    false,
    Style: tui.Style{
        Width:  tui.Cells(30),
        Height: tui.Cells(3),
        Border: tui.BorderAll,
    },
    FocusedStyle: tui.Style{
        Foreground: tui.ANSIColor(3),
    },
})
```

- Printable characters are appended to `Value` via `OnChange`.
- Backspace removes the last rune.
- Empty `Value` displays `Placeholder`.
- When focused, the terminal cursor is shown at the end of the text.
- `AutoFocus: true` — input receives focus on initial mount if nothing else is focused.
- `Disabled: true` — input cannot be focused or edited; no cursor is shown.
- `FocusedStyle` overlays visual properties only (foreground, background, bold,
  italic, underline). Layout fields (width, height, border, padding) are always
  taken from `Style`.

Current limitations: single-line only. Long values are clipped by the view bounds.
No cursor movement, selection, delete key, home/end, paste, or horizontal scrolling.

## Known limitations

- Mouse input is not supported.
- Text does not wrap; long lines are clipped to the content rect.
- Flexbox is a subset: `FlexGrow` is implemented. `FlexShrink` is reserved and
  has no effect. No `min-content` sizing, `flex-basis`, or wrapping.
- No async effects or hooks beyond `UseState` and `UseFocused`.
- Function component identity is based on the function pointer. Inline closures
  that change identity on every render may cause state to reset. Use named
  component functions, or wrap with `tui.Keyed` when component order may change.

## License

MIT
