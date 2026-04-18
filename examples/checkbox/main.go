// Command checkbox demonstrates the Checkbox widget.
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		checked, setChecked := tui.UseState(false)

		return tui.View(
			tui.Style{
				Padding:   tui.All(1),
				Direction: tui.Column,
				Gap:       1,
			},
			tui.Text("Checkbox Example", tui.WithTextStyle(tui.Style{Bold: true})),
			tui.Checkbox(tui.CheckboxProps{
				Label:    "Accept terms and conditions",
				Value:    checked,
				OnChange: setChecked,
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(0),
					Background: tui.ANSIColor(7),
				},
				AutoFocus: true,
			}),
			tui.View(
				tui.Style{Height: tui.Cells(1)},
				tui.Text(fmt.Sprintf("Status: %v", checked)),
			),
			tui.Text("Press Space or Enter to toggle. Press Ctrl-C to quit."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
