// Command twocounters demonstrates focus traversal with two independent counters.
// Tab switches focus between the counters; +/- changes only the focused one.
package main

import (
	"fmt"
	"os"
	"strconv"

	tui "github.com/smasonuk/earlgray"
)

// Counter is a component that displays a count and responds to +/- keys.
// It uses UseFocused to visually indicate which counter has focus.
func Counter() tui.Node {
	count, setCount := tui.UseState(0)
	focused := tui.UseFocused()

	label := "Count: " + strconv.Itoa(count)

	borderFg := tui.Color{}
	if focused {
		borderFg = tui.ANSIColor(3) // yellow when focused
	}

	return tui.ViewWith(
		tui.ViewProps{
			Style: tui.Style{
				Direction:  tui.Column,
				AlignItems: tui.AlignCenter,
				Justify:    tui.JustifyCenter,
				FlexGrow:   1,
			},
			OnKey: func(ev tui.KeyEvent) bool {
				switch ev.Rune {
				case '+':
					setCount(count + 1)
					return true
				case '-':
					setCount(count - 1)
					return true
				}
				return false
			},
			Focusable: true,
		},
		tui.View(
			tui.Style{
				Border:     tui.BorderAll,
				Padding:    tui.All(1),
				Foreground: borderFg,
			},
			tui.Text(label),
		),
		tui.View(
			tui.Style{Height: tui.Cells(1)},
			tui.Text(focusHint(focused)),
		),
	)
}

func focusHint(focused bool) string {
	if focused {
		return "[ focused ] +/- to change"
	}
	return "Tab to focus"
}

func App() tui.Node {
	return tui.View(
		tui.Style{Direction: tui.Row, FlexGrow: 1},
		tui.Keyed("a", tui.Component(Counter)),
		tui.Keyed("b", tui.Component(Counter)),
	)
}

func main() {
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
