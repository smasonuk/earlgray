// Command textpanel demonstrates the TextPanel widget.
package main

import (
	"fmt"
	"os"
	"strings"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		wrap, setWrap := tui.UseState(true)

		longText := strings.Join([]string{
			"Scrollable TextPanel",
			"",
			"This panel contains more text than fits on screen.",
			"Use Up/Down to scroll one line.",
			"Use PgUp/PgDown to scroll a page.",
			"Use Home/End to jump to the top or bottom.",
			"",
			"When word wrap is disabled, Left/Right scroll horizontally.",
			"",
			"Long line: abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 0123456789",
			"",
			strings.Repeat("More content. ", 20),
		}, "\n")

		return tui.ViewWith(
			tui.ViewProps{
				Style: tui.Style{
					Direction: tui.Column,
					Padding:   tui.All(1),
					Gap:       1,
					FlexGrow:  1,
				},
				OnKey: func(ev tui.KeyEvent) bool {
					if ev.Key == tui.KeyRune && ev.Rune == 'w' {
						setWrap(!wrap)
						return true
					}
					return false
				},
			},
			tui.Text("TextPanel Example", tui.WithTextStyle(tui.Style{Bold: true})),
			tui.Text(fmt.Sprintf("WordWrap: %v. Press 'w' to toggle. Ctrl-C quits.", wrap)),
			tui.TextPanel(tui.TextPanelProps{
				Text:          longText,
				WordWrap:      wrap,
				ShowScrollbar: true,
				AutoFocus:     true,
				Style: tui.Style{
					FlexGrow: 1,
					Border:   tui.BorderAll,
					Padding:  tui.All(1),
				},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(3),
				},
			}),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
