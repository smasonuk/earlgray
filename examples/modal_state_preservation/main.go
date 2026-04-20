package main

import (
	"log"

	tui "github.com/smasonuk/earlgray"
)

func main() {
	err := tui.Run(func() tui.Node {
		return tui.Component(func() tui.Node {
			appCtx := tui.UseApp()
			showModal, setShowModal := tui.UseState(false)
			inputText, setInputText := tui.UseState("")

			pageContent := tui.ViewWith(
				tui.ViewProps{
					Style: tui.Style{Direction: tui.Column, Padding: tui.All(2)},
				},
				tui.Text("Modal State Preservation Example", tui.WithTextStyle(tui.Style{Bold: true})),
				tui.Text(""),
				tui.Text("Type something below, then open the modal."),
				tui.Text("When you close it, the text should still be here."),
				tui.Text(""),
				tui.TextInput(tui.TextInputProps{
					Value:       inputText,
					OnChange:    setInputText,
					Placeholder: "Type here...",
					Style:       tui.Style{Border: tui.BorderAll},
					AutoFocus:   !showModal,
				}),
				tui.Text(""),
				tui.Button(tui.ButtonProps{
					Label:   "[Open Modal]",
					OnPress: func() { setShowModal(true) },
					Style:   tui.Style{Border: tui.BorderAll},
				}),
				tui.Text(""),
				tui.Text("Press Esc to quit."),
			)

			wrapped := tui.ViewWith(
				tui.ViewProps{
					Style:     tui.Style{FlexGrow: 1},
					AutoFocus: true,
					OnKey: func(ev tui.KeyEvent) bool {
						if ev.Key == tui.KeyEsc && !showModal {
							appCtx.Quit()
							return true
						}
						return false
					},
				},
				pageContent,
			)

			if !showModal {
				return tui.Overlay(wrapped)
			}

			modal := tui.Dialog(
				tui.DialogProps{
					Style: tui.Style{
						Border:     tui.BorderAll,
						Foreground: tui.ANSIColor(62),
						Padding:    tui.All(1),
						Width:      tui.Cells(40),
					},
					BackdropStyle: tui.Style{Background: tui.ANSIColor(0)},
					CloseOnEsc:    true,
					OnClose:       func() { setShowModal(false) },
				},
				tui.ViewWith(
					tui.ViewProps{
						Style:     tui.Style{Direction: tui.Column},
						AutoFocus: true,
						OnKey: func(ev tui.KeyEvent) bool {
							if ev.Key == tui.KeyEnter {
								setShowModal(false)
								return true
							}
							return false
						},
					},
					tui.Text("MODAL", tui.WithTextStyle(tui.Style{Bold: true})),
					tui.Text(""),
					tui.Text("This is a modal overlay."),
					tui.Text(""),
					tui.Button(tui.ButtonProps{
						Label:     "[Close Modal]",
						OnPress:   func() { setShowModal(false) },
						Style:     tui.Style{Border: tui.BorderAll},
						AutoFocus: true,
					}),
				),
			)

			return tui.Overlay(wrapped, modal)
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
