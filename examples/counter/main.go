// Command counter demonstrates UseState for interactive state management.
package main

import (
	"fmt"
	"os"
	"strconv"

	tui "github.com/smason/earlgray"
)

// Counter is a component that displays a count and responds to +/- keys.
// Note: key handling is not yet wired to components, so this demonstrates
// the UseState API and renders the initial state.
func Counter() tui.Node {
	count, _ := tui.UseState(0)

	label := "Counter, i hardly know her: " + strconv.Itoa(count)

	return tui.View(
		tui.Style{
			Direction:  tui.Column,
			AlignItems: tui.AlignCenter,
			Justify:    tui.JustifyCenter,
			FlexGrow:   1,
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
			tui.Text("Press 'q' to quit"),
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
