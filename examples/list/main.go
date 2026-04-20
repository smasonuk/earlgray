// Command list demonstrates the List widget.
package main

import (
	"fmt"
	"os"

	tui "github.com/smasonuk/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		items := []string{"Apples", "Bananas", "Cherries", "Dates", "Elderberries"}
		selected, setSelected := tui.UseState(0)

		return tui.View(
			tui.Style{
				Padding:   tui.All(1),
				Direction: tui.Column,
				Gap:       1,
			},
			tui.Text("List Example", tui.WithTextStyle(tui.Style{Bold: true})),
			tui.List(tui.ListProps{
				Items:         items,
				SelectedIndex: selected,
				OnSelect:      setSelected,
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(0),
					Background: tui.ANSIColor(7),
				},
				AutoFocus: true,
			}),
			tui.View(
				tui.Style{Height: tui.Cells(1)},
				tui.Text(fmt.Sprintf("Selected: %s", items[selected])),
			),
			tui.Text("Use Up/Down keys to navigate. Press Ctrl-C to quit."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
