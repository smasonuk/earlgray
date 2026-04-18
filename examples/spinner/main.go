// Command spinner demonstrates Spinner with a goroutine-driven background task.
package main

import (
	"fmt"
	"os"
	"time"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		loading, setLoading := tui.UseState(true)
		message, setMessage := tui.UseState("Starting background task...")

		tui.UseEffect(func() func() {
			stop := make(chan struct{})

			go func() {
				select {
				case <-time.After(2 * time.Second):
					setMessage("Background task finished.")
					setLoading(false)
				case <-stop:
					return
				}
			}()

			return func() {
				close(stop)
			}
		})

		content := tui.Text(message)
		if loading {
			content = tui.Spinner(tui.SpinnerProps{
				Active: true,
				Label:  message,
				Style: tui.Style{
					Foreground: tui.ANSIColor(6),
					Bold:       true,
				},
			})
		}

		return tui.View(
			tui.Style{
				Direction:  tui.Column,
				Justify:    tui.JustifyCenter,
				AlignItems: tui.AlignCenter,
				FlexGrow:   1,
				Padding:    tui.All(1),
				Gap:        1,
			},
			tui.Text("Spinner Example", tui.WithTextStyle(tui.Style{Bold: true})),
			content,
			tui.Text("This finishes automatically after 2 seconds."),
			tui.Text("Press Ctrl-C to quit."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
