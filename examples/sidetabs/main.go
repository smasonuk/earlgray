// Command sidetabs demonstrates the SideTabs widget.
package main

import (
	"fmt"
	"os"

	tui "github.com/smasonuk/earlgray"
)

func App() tui.Node {
	return tui.Component(func() tui.Node {
		active, setActive := tui.UseState("overview")

		return tui.SideTabs(tui.SideTabsProps{
			Value:     active,
			OnChange:  setActive,
			AutoFocus: true,
			Style: tui.Style{
				FlexGrow: 1,
				Padding:  tui.All(1),
			},
			TabListStyle: tui.Style{
				Border:  tui.BorderRight,
				Padding: tui.All(1),
			},
			PanelStyle: tui.Style{
				FlexGrow: 1,
				Padding:  tui.All(1),
			},
			FocusedTabStyle: tui.Style{
				Foreground: tui.ANSIColor(3),
				Bold:       true,
			},
			Tabs: []tui.SideTab{
				{
					Label: "Overview",
					Value: "overview",
					Content: tui.View(
						tui.Style{Direction: tui.Column, Gap: 1},
						tui.Text("Overview", tui.WithTextStyle(tui.Style{Bold: true})),
						tui.Text("System status: healthy"),
						tui.Text("Active jobs: 4"),
					),
				},
				{
					Label: "Settings",
					Value: "settings",
					Content: tui.View(
						tui.Style{Direction: tui.Column, Gap: 1},
						tui.Text("Settings", tui.WithTextStyle(tui.Style{Bold: true})),
						tui.Checkbox(tui.CheckboxProps{Label: "Enable notifications", Value: true}),
						tui.Select(tui.SelectProps{
							Options: []tui.RadioOption{
								{Label: "Light", Value: "light"},
								{Label: "Dark", Value: "dark"},
							},
							Value: "light",
						}),
					),
				},
				{
					Label: "Logs",
					Value: "logs",
					Content: tui.View(
						tui.Style{Direction: tui.Column, Gap: 1},
						tui.Text("Logs", tui.WithTextStyle(tui.Style{Bold: true})),
						tui.Text("10:12 worker started"),
						tui.Text("10:13 cache warmed"),
						tui.Text("10:14 ready"),
					),
				},
			},
		})
	})
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
