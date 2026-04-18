// Command basic demonstrates a header/sidebar/content layout.
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

// App renders a classic three-pane layout: header, sidebar, main content.
func App() tui.Node {
	return tui.View(
		tui.Style{Direction: tui.Column, FlexGrow: 1},
		// Header bar.
		tui.View(
			tui.Style{Height: tui.Cells(1), Border: tui.BorderBottom, Background: tui.ANSIColor(0)},
			tui.Text("  EarlGray TUI  "),
		),
		// Body row: sidebar + main.
		tui.View(
			tui.Style{Direction: tui.Row, FlexGrow: 1},
			// Sidebar.
			tui.View(
				tui.Style{Width: tui.Cells(24), Border: tui.BorderRight},
				tui.Text("Navigation", tui.WithAlign(tui.AlignStart)),
			),
			// Main content.
			tui.View(
				tui.Style{FlexGrow: 1, Padding: tui.All(1)},
				tui.Text("Main content area"),
			),
		),
	)
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
