// Command counter demonstrates UseState for interactive state management.
package main

import (
	"fmt"
	"os"
	"strconv"

	tui "github.com/smason/earlgray"
)

// Counter is a component that displays a count and responds to +/- keys.
func Counter() tui.Node {
	count, setCount := tui.UseState(0)

	label := "Count: " + strconv.Itoa(count)

	return tui.ViewWith(
		tui.ViewProps{
			Style: tui.Style{
				Direction:  tui.Column,
				AlignItems: tui.AlignCenter,
				Justify:    tui.JustifyCenter,
				FlexGrow:   1,
			},
			OnKey: func(ev tui.KeyEvent) bool {
				switch ev.Rune {
				case '+':
					setCount(count + 1)
					return true
				case '-':
					setCount(count - 1)
					return true
				}
				return false
			},
		},
		tui.View(
			tui.Style{
				Border:  tui.BorderAll,
				Padding: tui.All(1),
			},
			tui.Text(label),
		),
		tui.View(
			tui.Style{Height: tui.Cells(1), Padding: tui.Insets{Left: 1}},
			tui.Text("Press '+'/'-' to change, Ctrl-C to quit"),
		),
	)
}

func App() tui.Node {
	return tui.Component(Counter)
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
