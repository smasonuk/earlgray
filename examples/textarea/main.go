// Command textarea demonstrates the TextArea widget with controlled state.
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		value, setValue := tui.UseState("")
		submitted, setSubmitted := tui.UseState("")

		return tui.View(
			tui.Style{
				Direction: tui.Column,
				Padding:   tui.All(1),
				Gap:       1,
				FlexGrow:  1,
			},
			tui.Text("TextArea Example", tui.WithTextStyle(tui.Style{Bold: true})),
			tui.TextArea(tui.TextAreaProps{
				Value:             value,
				OnChange:          setValue,
				OnSubmit:          setSubmitted,
				Placeholder:       "Write multiple lines here...",
				ShowScrollbar:     true,
				SubmitOnCtrlEnter: true,
				AutoFocus:         true,
				Style: tui.Style{
					Width:   tui.Cells(50),
					Height:  tui.Cells(10),
					Border:  tui.BorderAll,
					Padding: tui.All(1),
				},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(3),
				},
			}),
			tui.Text("Submitted with Ctrl+Enter:"),
			tui.Text(submitted),
			tui.Text("Help: Enter inserts newline. Ctrl+Enter submits. Up/Down scroll through wrapped visual lines. Ctrl-C quits."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
