// Command button demonstrates the Button widget with focus and event handling.
package main

import (
	"fmt"
	"os"
	"strconv"

	tui "github.com/smason/earlgray"
)

// ButtonCounter is a component demonstrating Button widgets.
func ButtonCounter() tui.Node {
	count, setCount := tui.UseState(0)

	return tui.View(
		tui.Style{
			Direction:  tui.Column,
			AlignItems: tui.AlignCenter,
			Justify:    tui.JustifyCenter,
			FlexGrow:   1,
			Padding:    tui.All(1),
		},
		// Counter display
		tui.View(
			tui.Style{
				Direction: tui.Row,
				Gap:       2,
				Padding:   tui.All(1),
				Border:    tui.BorderAll,
			},
			tui.Text("Count:", tui.WithTextStyle(tui.Style{FlexShrink: 1})),
			tui.Text(strconv.Itoa(count), tui.WithTextStyle(tui.Style{FlexShrink: 1, Bold: true})),
		),
		// Spacer
		tui.View(tui.Style{Height: tui.Cells(1)}),
		// Buttons
		tui.View(
			tui.Style{
				Direction: tui.Row,
				Gap:       2,
			},
			tui.Button(tui.ButtonProps{
				Label: "[ - ]",
				OnPress: func() {
					setCount(count - 1)
				},
				Style: tui.Style{
					Width:   tui.Cells(7),
					Height:  tui.Cells(3),
					Border:  tui.BorderAll,
				},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(3), // yellow
				},
			}),
			tui.Button(tui.ButtonProps{
				Label: "[ + ]",
				OnPress: func() {
					setCount(count + 1)
				},
				Style: tui.Style{
					Width:   tui.Cells(7),
					Height:  tui.Cells(3),
					Border:  tui.BorderAll,
				},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(3), // yellow
				},
			}),
		),
		// Help text
		tui.View(
			tui.Style{Height: tui.Cells(1)},
			tui.Text("Tab to switch, Enter/Space to press, Ctrl-C to quit", tui.WithTextStyle(tui.Style{FlexShrink: 1})),
		),
	)
}

func App() tui.Node {
	return tui.Component(ButtonCounter)
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
