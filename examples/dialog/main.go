// Command dialog demonstrates modal dialogs, focus trapping, and focus restoration.
package main

import (
	"fmt"
	"os"

	tui "github.com/smasonuk/earlgray"
)

func App() tui.Node {
	return tui.Component(AppRoot)
}

func AppRoot() tui.Node {
	open, setOpen := tui.UseState(false)
	submitted, setSubmitted := tui.UseState("")

	page := tui.View(
		tui.Style{
			Direction: tui.Column,
			Padding:   tui.All(1),
			Gap:       1,
			FlexGrow:  1,
		},
		tui.Text("Dialog Example", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.Text("Tab between buttons. Press Enter or Space to activate."),
		tui.Button(tui.ButtonProps{
			Label: "Other button",
			OnPress: func() {
				setSubmitted("Other button pressed")
			},
			Style: tui.Style{
				Width:  tui.Cells(24),
				Height: tui.Cells(3),
				Border: tui.BorderAll,
			},
			FocusedStyle: tui.Style{
				Foreground: tui.ANSIColor(3),
			},
		}),
		tui.Button(tui.ButtonProps{
			Label: "Open dialog",
			OnPress: func() {
				setOpen(true)
			},
			Style: tui.Style{
				Width:  tui.Cells(24),
				Height: tui.Cells(3),
				Border: tui.BorderAll,
			},
			FocusedStyle: tui.Style{
				Foreground: tui.ANSIColor(3),
			},
		}),
		tui.Text("Submitted: "+submitted),
		tui.Text("Ctrl-C quits."),
	)

	if !open {
		return page
	}

	return tui.Overlay(
		page,
		tui.Dialog(tui.DialogProps{
			CloseOnEsc: true,
			OnClose: func() {
				setOpen(false)
			},
			BackdropStyle: tui.Style{
				Background: tui.ANSIColor(0),
			},
			Style: tui.Style{
				Width:   tui.Cells(50),
				Height:  tui.Cells(15),
				Border:  tui.BorderAll,
				Padding: tui.All(1),
			},
		}, dialogContent(setOpen, setSubmitted)),
	)
}

func dialogContent(setOpen func(bool), setSubmitted func(string)) tui.Node {
	return tui.Component(func() tui.Node {
		value, setValue := tui.UseState("")

		return tui.View(
			tui.Style{
				Direction: tui.Column,
				Gap:       1,
			},
			tui.Text("Enter a value:"),
			tui.TextInput(tui.TextInputProps{
				Value:       value,
				OnChange:    setValue,
				Placeholder: "Type here...",
				AutoFocus:   true,
				Style: tui.Style{
					Width:  tui.Cells(36),
					Height: tui.Cells(3),
					Border: tui.BorderAll,
				},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(3),
				},
				OnSubmit: func(s string) {
					setSubmitted(s)
					setOpen(false)
				},
			}),
			tui.View(
				tui.Style{
					Direction: tui.Row,
					Gap:       2,
				},
				tui.Button(tui.ButtonProps{
					Label: "Submit",
					OnPress: func() {
						setSubmitted(value)
						setOpen(false)
					},
					Style: tui.Style{
						Width:  tui.Cells(12),
						Height: tui.Cells(3),
						Border: tui.BorderAll,
					},
					FocusedStyle: tui.Style{
						Foreground: tui.ANSIColor(3),
					},
				}),
				tui.Button(tui.ButtonProps{
					Label: "Cancel",
					OnPress: func() {
						setOpen(false)
					},
					Style: tui.Style{
						Width:  tui.Cells(12),
						Height: tui.Cells(3),
						Border: tui.BorderAll,
					},
					FocusedStyle: tui.Style{
						Foreground: tui.ANSIColor(3),
					},
				}),
			),
			tui.Text("Esc cancels. Enter submits from the input."),
		)
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
