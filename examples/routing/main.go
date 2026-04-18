package main

import (
	"fmt"
	"log"

	tui "github.com/smason/earlgray"
)

func main() {
	if err := tui.Run(App); err != nil {
		log.Fatal(err)
	}
}

func App() tui.Node {
	return tui.Component(AppRoot)
}

func AppRoot() tui.Node {
	router := tui.UseRouter("home")
	modalOpen, setModalOpen := tui.UseState(false)
	submittedValue, setSubmittedValue := tui.UseState("")

	var page tui.Node
	switch router.Path {
	case "settings":
		page = SettingsPage(router, setModalOpen)
	default:
		page = HomePage(router, setModalOpen, submittedValue)
	}

	if modalOpen {
		return tui.Overlay(
			page,
			tui.Dialog(tui.DialogProps{
				CloseOnEsc: true,
				OnClose: func() {
					setModalOpen(false)
				},
				BackdropStyle: tui.Style{Background: tui.ANSIColor(0)},
				Style: tui.Style{
					Width:  tui.Cells(40),
					Height: tui.Cells(10),
					Border: tui.BorderAll,
				},
			}, ModalContent(setModalOpen, setSubmittedValue)),
		)
	}

	return page
}

func HomePage(router tui.Router, setModalOpen func(bool), submitted string) tui.Node {
	status := "No value submitted yet."
	if submitted != "" {
		status = fmt.Sprintf("Last submitted: %s", submitted)
	}

	return tui.ViewWith(
		tui.ViewProps{
			Style: tui.Style{
				FlexGrow:  1,
				Direction: tui.Column,
				Padding:   tui.All(1),
				Gap:       1,
			},
			OnKey: func(ev tui.KeyEvent) bool {
				if ev.Key == tui.KeyRune {
					switch ev.Rune {
					case 's':
						router.Push("settings")
						return true
					case 'm':
						setModalOpen(true)
						return true
					}
				}
				return false
			},
		},
		tui.Text("HOME PAGE", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.Text("Press 's' for Settings, 'm' for Modal"),
		tui.Text(status),
	)
}

func SettingsPage(router tui.Router, setModalOpen func(bool)) tui.Node {
	return tui.ViewWith(
		tui.ViewProps{
			Style: tui.Style{
				FlexGrow:  1,
				Direction: tui.Column,
				Padding:   tui.All(1),
				Gap:       1,
			},
			OnKey: func(ev tui.KeyEvent) bool {
				if ev.Key == tui.KeyRune {
					switch ev.Rune {
					case 'h':
						router.Back()
						return true
					case 'm':
						setModalOpen(true)
						return true
					}
				}
				return false
			},
		},
		tui.Text("SETTINGS PAGE", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.Text("Press 'h' for Home, 'm' for Modal"),
	)
}

func ModalContent(setModalOpen func(bool), setSubmittedValue func(string)) tui.Node {
	return tui.Component(func() tui.Node {
		val, setVal := tui.UseState("")

		return tui.ViewWith(
			tui.ViewProps{
				Style: tui.Style{
					Direction: tui.Column,
					Padding:   tui.All(1),
					Gap:       1,
				},
			},
			tui.Text("Enter some text:"),
			tui.TextInput(tui.TextInputProps{
				Value:     val,
				AutoFocus: true,
				OnChange:  setVal,
				OnSubmit: func(s string) {
					setSubmittedValue(s)
					setModalOpen(false)
				},
				Style: tui.Style{
					Border: tui.BorderAll,
				},
			}),
			tui.View(
				tui.Style{Direction: tui.Row, Gap: 2},
				tui.Button(tui.ButtonProps{
					Label: "Submit",
					OnPress: func() {
						setSubmittedValue(val)
						setModalOpen(false)
					},
				}),
				tui.Button(tui.ButtonProps{
					Label: "Cancel",
					OnPress: func() {
						setModalOpen(false)
					},
				}),
			),
		)
	})
}
