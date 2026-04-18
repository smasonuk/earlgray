// Command kitchensink demonstrates the main EarlGray public API together:
// UseState, UseRouter, Button, TextInput, Checkbox, List, Select, TextPanel,
// Dialog, Overlay, FocusedStyle, and Keyed.
//
// Keyboard shortcuts:
//
//	Tab / Shift+Tab  change focus
//	Enter / Space    activate buttons and controls
//	Esc              close dialog
//	Ctrl-C           quit
package main

import (
	"fmt"
	"os"

	tui "github.com/smason/earlgray"
)

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func App() tui.Node {
	return tui.Component(AppRoot)
}

func AppRoot() tui.Node {
	router := tui.UseRouter("home")
	dialogOpen, setDialogOpen := tui.UseState(false)
	lastSubmit, setLastSubmit := tui.UseState("")

	var page tui.Node
	switch router.Path {
	case "settings":
		page = SettingsPage(router)
	default:
		page = HomePage(router, setDialogOpen, lastSubmit)
	}

	if dialogOpen {
		return tui.Overlay(
			page,
			tui.Dialog(tui.DialogProps{
				CloseOnEsc: true,
				OnClose:    func() { setDialogOpen(false) },
				Style: tui.Style{
					Width:  tui.Cells(50),
					Height: tui.Cells(10),
					Border: tui.BorderAll,
				},
			}, DialogContent(setDialogOpen, setLastSubmit)),
		)
	}

	return page
}

// HomePage is the main landing page.
func HomePage(router tui.Router, setDialogOpen func(bool), lastSubmit string) tui.Node {
	status := "No value submitted yet."
	if lastSubmit != "" {
		status = "Last submitted: " + lastSubmit
	}

	return tui.ViewWith(
		tui.ViewProps{
			Style: tui.Style{
				Direction: tui.Column,
				FlexGrow:  1,
				Padding:   tui.All(1),
				Gap:       1,
			},
		},
		tui.Text("KITCHEN SINK DEMO", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.Text("Tab/Shift+Tab: focus  Enter/Space: activate  Ctrl-C: quit"),
		tui.View(tui.Style{Direction: tui.Row, Gap: 2},
			tui.Keyed("open-dialog", tui.Button(tui.ButtonProps{
				Label:        "[ Open Dialog ]",
				FocusedStyle: tui.Style{Bold: true, Foreground: tui.ANSIColor(3)},
				OnPress:      func() { setDialogOpen(true) },
			})),
			tui.Keyed("go-settings", tui.Button(tui.ButtonProps{
				Label:        "[ Settings ]",
				FocusedStyle: tui.Style{Bold: true, Foreground: tui.ANSIColor(2)},
				OnPress:      func() { router.Push("settings") },
			})),
		),
		tui.Text(status),
	)
}

// SettingsPage shows a collection of controls for demo purposes.
func SettingsPage(router tui.Router) tui.Node {
	return tui.Component(SettingsPageRoot(router))
}

func SettingsPageRoot(router tui.Router) func() tui.Node {
	return func() tui.Node {
		name, setName := tui.UseState("")
		enabled, setEnabled := tui.UseState(false)
		theme, setTheme := tui.UseState("light")
		selectedItem, setSelectedItem := tui.UseState(0)

		helpText := "Settings page help\n\nUse Tab to move between controls.\n" +
			"TextInput: type to edit.\nCheckbox: Space/Enter to toggle.\n" +
			"Select: Left/Right to change.\nList: Up/Down to navigate.\n\n" +
			"Press the Back button to return home."

		return tui.ViewWith(
			tui.ViewProps{
				Style: tui.Style{
					Direction: tui.Column,
					FlexGrow:  1,
					Padding:   tui.All(1),
					Gap:       1,
				},
			},
			tui.Text("SETTINGS", tui.WithTextStyle(tui.Style{Bold: true})),
			// Name input
			tui.View(tui.Style{Direction: tui.Row, Gap: 1},
				tui.Text("Name:"),
				tui.Keyed("name-input", tui.TextInput(tui.TextInputProps{
					Value:        name,
					OnChange:     setName,
					Placeholder:  "enter name",
					Style:        tui.Style{Width: tui.Cells(20), Border: tui.BorderAll},
					FocusedStyle: tui.Style{Foreground: tui.ANSIColor(3)},
				})),
			),
			// Checkbox
			tui.Keyed("enabled-check", tui.Checkbox(tui.CheckboxProps{
				Label:    "Enable feature",
				Value:    enabled,
				OnChange: setEnabled,
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(0),
					Background: tui.ANSIColor(7),
				},
			})),
			// Select
			tui.View(tui.Style{Direction: tui.Row, Gap: 1},
				tui.Text("Theme:"),
				tui.Keyed("theme-select", tui.Select(tui.SelectProps{
					Options: []tui.RadioOption{
						{Label: "Light", Value: "light"},
						{Label: "Dark", Value: "dark"},
						{Label: "System", Value: "system"},
					},
					Value:    theme,
					OnChange: setTheme,
					Style:    tui.Style{Width: tui.Cells(14)},
					FocusedStyle: tui.Style{
						Foreground: tui.ANSIColor(0),
						Background: tui.ANSIColor(7),
					},
				})),
			),
			// List
			tui.Text("Items:"),
			tui.Keyed("item-list", tui.List(tui.ListProps{
				Items:         []string{"Alpha", "Beta", "Gamma", "Delta"},
				SelectedIndex: selectedItem,
				OnSelect:      setSelectedItem,
				Style:         tui.Style{Height: tui.Cells(6), Border: tui.BorderAll},
				FocusedStyle:  tui.Style{Foreground: tui.ANSIColor(6)},
			})),
			// TextPanel with scrollable help text
			tui.Keyed("help-panel", tui.TextPanel(tui.TextPanelProps{
				Text:          helpText,
				Style:         tui.Style{Height: tui.Cells(5), Border: tui.BorderAll},
				FocusedStyle:  tui.Style{Foreground: tui.ANSIColor(2)},
				WordWrap:      true,
				ShowScrollbar: true,
			})),
			// Status line
			tui.Text(fmt.Sprintf("name=%q  enabled=%v  theme=%s  item=%d",
				name, enabled, theme, selectedItem)),
			// Back button
			tui.Keyed("back-btn", tui.Button(tui.ButtonProps{
				Label: "[ Back ]",
				Style: tui.Style{Width: tui.Cells(10), Height: tui.Cells(1)},
				FocusedStyle: tui.Style{
					Foreground: tui.ANSIColor(0),
					Background: tui.ANSIColor(7),
					Bold:       true,
				},
				OnPress: func() { router.Back() },
			})),
		)
	}
}

// DialogContent is the content rendered inside the dialog.
func DialogContent(setDialogOpen func(bool), setLastSubmit func(string)) tui.Node {
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
			tui.Text("Enter a value (Esc to close):"),
			tui.Keyed("dialog-input", tui.TextInput(tui.TextInputProps{
				Value:     val,
				AutoFocus: true,
				OnChange:  setVal,
				OnSubmit: func(s string) {
					setLastSubmit(s)
					setDialogOpen(false)
				},
				Style:        tui.Style{Border: tui.BorderAll},
				FocusedStyle: tui.Style{Foreground: tui.ANSIColor(3)},
			})),
			tui.View(tui.Style{Direction: tui.Row, Gap: 2},
				tui.Keyed("submit-btn", tui.Button(tui.ButtonProps{
					Label:        "[ Submit ]",
					FocusedStyle: tui.Style{Bold: true, Foreground: tui.ANSIColor(2)},
					OnPress: func() {
						setLastSubmit(val)
						setDialogOpen(false)
					},
				})),
				tui.Keyed("cancel-btn", tui.Button(tui.ButtonProps{
					Label:        "[ Cancel ]",
					FocusedStyle: tui.Style{Bold: true},
					OnPress:      func() { setDialogOpen(false) },
				})),
			),
		)
	})
}
