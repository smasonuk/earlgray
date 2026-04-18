# EarlGray Architecture

EarlGray is a React-like retained-mode TUI library in Go backed by `tcell`.

## Layer diagram

```
┌─────────────────────────────────┐
│         Public API (tui.go)     │  View, Text, Keyed, Component, UseState, Run
├─────────────────────────────────┤
│         Runtime (internal/runtime)│  Instance tree, reconciliation, hooks
├──────────────┬──────────────────┤
│  Layout      │  Render          │  Flex layout engine; cell-buffer painter
│(internal/layout)│(internal/render+runtime)│
├──────────────┴──────────────────┤
│  Screen (internal/screen)       │  2D cell buffer + diff engine
├─────────────────────────────────┤
│  Host (internal/host)           │  tcell backend abstraction
└─────────────────────────────────┘
```

## Package responsibilities

| Package | Responsibility |
|---------|---------------|
| `tui` (root) | Public API surface: constructors, Run loop, type aliases |
| `internal/color` | Shared Color type with tcell conversion |
| `internal/style` | Style, Rect, Insets, Dimension, Border primitives |
| `internal/node` | Node tree types (ViewKind, TextKind, etc.) |
| `internal/event` | Unified internal event types |
| `internal/screen` | Cell buffer, SetCell, DrawHLine/VLine, Diff |
| `internal/layout` | Flex layout engine: Constraints → Tree of Rects |
| `internal/runtime` | Retained Instance tree, reconciliation, UseState hooks |
| `internal/render` | FlushDiff helper wrapping screen.Diff |
| `internal/host` | Host interface + TcellHost implementation |
| `internal/focus` | Focus tracking by element ID |
| `internal/testutil` | Buffer-to-grid helpers for tests |

## Data flow

```
root() → Node tree
         ↓ runtime.Update() reconciles → Instance tree
         ↓ runtime.RunLayout() → layout.Layout() → Tree of Rects
         ↓ runtime.Render() → screen.Buffer (cell grid)
         ↓ render.FlushDiff(prev, next) → host.SetCell() per changed cell
         ↓ host.Show() → terminal
```

## Rendering model

- **Cell buffer**: a flat `[]Cell` (rune + style) of size W×H, row-major.
- **Diff**: compare previous and current buffers cell-by-cell; only changed cells are sent to tcell.
- **No ANSI string composition**: all drawing goes through `Buffer.SetCell` and then `tcell.Screen.SetContent`.

## Reconciliation

Inspired by React's Fiber reconciler (simplified):

1. Match children by explicit key (if set) then by position + kind.
2. Same key/position + same kind → reuse Instance, update node descriptor.
3. Different kind or no match → create new Instance (hook slots reset).
4. Component instances call the render function with `renderingInstance` set so `UseState` can find the right slot.

## Layout algorithm (flex subset)

1. Resolve fixed-size children (DimCells).
2. Calculate remaining space after fixed children + gaps.
3. Distribute remaining space proportionally to FlexGrow children.
4. Apply JustifyContent offsets (start/center/end/space-between).
5. Apply AlignItems on the cross axis per child.
6. Position children absolutely within parent content rect.
