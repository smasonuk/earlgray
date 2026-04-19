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
		copied, setCopied := tui.UseState("")

		app := tui.UseApp()

		return tui.ViewWith(
			tui.ViewProps{
				Style: tui.Style{
					Direction: tui.Column,
					Padding:   tui.All(1),
					Gap:       1,
					FlexGrow:  1,
				},
				OnKeyCapture: func(ev tui.KeyEvent) bool {
					if ev.Key == tui.KeyCtrlC && ev.Rune == 0 {
						// Ctrl+C is handled by the textarea OnCopy hook; don't quit here.
						return false
					}
					if ev.Key == tui.KeyRune && ev.Rune == 'q' && ev.Mod&tui.ModCtrl != 0 {
						app.Quit()
						return true
					}
					return false
				},
			},
			tui.Text("TextArea Example", tui.WithTextStyle(tui.Style{Bold: true})),
			tui.TextArea(tui.TextAreaProps{
				Value:             value,
				OnChange:          setValue,
				OnSubmit:          setSubmitted,
				OnCopy:            setCopied,
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
			tui.Text("Copied selection:"),
			tui.Text(copied),
			tui.Text("Help: Enter inserts newline. Ctrl+Enter submits. Shift+Arrows select. Ctrl+C copies selection. Ctrl+Q quits."),
		)
	})
}

func main() {
	if err := tui.RunWithOptions(App, tui.RunOptions{
		DisableCtrlCQuit: true,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
