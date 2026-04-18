# EarlGray

A React-like retained-mode TUI library for Go, backed by [tcell](https://github.com/gdamore/tcell).

## Features

- **Function components** — write components as plain Go functions returning `tui.Node`.
- **UseState** — React-like state hook; state persists across renders.
- **Flexbox layout subset** — row/column direction, fixed sizes, flex-grow, padding, gap, border, alignment.
- **Cell-buffer rendering** — diff-based: only changed terminal cells are flushed.
- **tcell backend** — full terminal compatibility via tcell v2.

## Quick start

```go
package main

import (
    "fmt"
    "os"

    "github.com/smason/earlgray"
)

func App() tui.Node {
    return tui.View(
        tui.Style{Direction: tui.Column, FlexGrow: 1},
        tui.View(
            tui.Style{Height: tui.Cells(1), Border: tui.BorderBottom},
            tui.Text("Header"),
        ),
        tui.View(
            tui.Style{Direction: tui.Row, FlexGrow: 1},
            tui.View(
                tui.Style{Width: tui.Cells(24), Border: tui.BorderRight},
                tui.Text("Sidebar"),
            ),
            tui.View(
                tui.Style{FlexGrow: 1, Padding: tui.All(1)},
                tui.Text("Main content"),
            ),
        ),
    )
}

func main() {
    if err := tui.Run(App); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Press `q` or `Ctrl-C` to quit.

## State management

```go
func Counter() tui.Node {
    count, setCount := tui.UseState(0)
    _ = setCount // wire to key events
    return tui.Text(fmt.Sprintf("Count: %d", count))
}

func App() tui.Node {
    return tui.Component(Counter)
}
```

## API reference

| Function | Description |
|----------|-------------|
| `Run(root func() Node) error` | Start the TUI main loop |
| `View(style Style, children ...Node) Node` | Container node |
| `Text(value string, opts ...TextOption) Node` | Text leaf node |
| `Keyed(key string, child Node) Node` | Stable reconciliation key |
| `Component(fn func() Node) Node` | Wrap a function component |
| `UseState[T](initial T) (T, func(T))` | State hook for components |
| `Cells(n int) Dimension` | Fixed cell-count dimension |
| `Auto() Dimension` | Auto/fill dimension |
| `All(n int) Insets` | Uniform padding/margin |

## License

MIT
