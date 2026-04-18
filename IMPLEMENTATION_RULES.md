# Implementation Rules

These constraints must never be violated when contributing to EarlGray.

## Hard constraints

1. **No Bubble Tea** — do not import `github.com/charmbracelet/bubbletea`.
2. **No tview** — do not import `github.com/rivo/tview`.
3. **No Lip Gloss as renderer** — termenv/lipgloss may be used for color profile detection only; final drawing always goes through tcell.
4. **No ANSI string rendering** — never compose multi-line strings and write them to the terminal. All rendering must go through `Buffer.SetCell` → `host.SetCell` → `tcell.Screen.SetContent`.
5. **No full CSS flexbox** — only row/column direction, fixed sizes, flex-grow, padding, gap, border, alignment. No flex-shrink algorithm, no wrapping, no grid.
6. **No mouse, scroll, animation, or advanced widgets** — the library is intentionally minimal.

## Architecture rules

- `internal/*` packages must not import the root `tui` package (avoids circular deps).
- Public types (`Style`, `Color`, etc.) live in the root package as type aliases over `internal/style` and `internal/color`.
- `internal/runtime` is the only package that mutates Instance state; all other packages are pure functions or data types.
- Rendering is single-threaded; `renderingInstance` global is safe.
- `UseState` panics if called outside a component render.

## Code style

- All exported symbols must have doc comments.
- Tests must use exact rect assertions (`style.Rect{X:…}`) not approximate ones.
- Test helper functions in `testutil` must be used for buffer inspection in integration-style tests.
