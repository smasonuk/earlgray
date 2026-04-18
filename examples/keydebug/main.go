// Command keydebug shows the last normalized key event EarlGray received.
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		last, setLast := tui.UseState("Press keys...")

		return tui.ViewWith(
			tui.ViewProps{
				Style: tui.Style{
					FlexGrow: 1,
					Padding:  tui.All(1),
				},
				Focusable: true,
				AutoFocus: true,
				OnKey: func(ev tui.KeyEvent) bool {
					setLast(fmt.Sprintf("Key=%v Rune=%q Mod=%v", ev.Key, ev.Rune, ev.Mod))
					return true
				},
			},
			tui.Text(last),
			tui.Text("Ctrl-I may appear as Tab depending on your terminal."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
