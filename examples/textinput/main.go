// Command textinput demonstrates the TextInput widget with controlled state.
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

func App() tui.Node {
	return tui.Component(Form)
}

func Form() tui.Node {
	name, setName := tui.UseState("")
	submitted, setSubmitted := tui.UseState("")

	greeting := "Hello"
	if name != "" {
		greeting = "Hello, " + name
	}

	return tui.View(
		tui.Style{
			Direction: tui.Column,
			Padding:   tui.All(1),
			Gap:       1,
			FlexGrow:  1,
		},
		tui.Text("Name:"),
		tui.TextInput(tui.TextInputProps{
			Value:       name,
			OnChange:    setName,
			OnSubmit:    func(v string) { setSubmitted(v) },
			Placeholder: "Type your name...",
			Style: tui.Style{
				Width:  tui.Cells(30),
				Height: tui.Cells(3),
				Border: tui.BorderAll,
			},
			FocusedStyle: tui.Style{
				Foreground: tui.ANSIColor(3),
			},
		}),
		tui.Text(greeting),
		tui.Text("Submitted: "+submitted),
		tui.Text("Help:"),
		tui.Text("Type to edit. Left/Right/Home/End move cursor. Enter submits. Ctrl-C quits."),
	)
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
