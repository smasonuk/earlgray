package main

import (
	"fmt"
	"os"

	tui "github.com/smasonuk/earlgray"
)

func App() tui.Node {
	copied, setCopied := tui.UseState("")

	placeholderText := `This is a ReadOnly TextArea example.
It demonstrates a selectable text panel mode.
You can drag to select this text, and it will auto-copy on release.

Typing, deleting, pasting, and pressing Enter are disabled in this mode.
The cursor is also hidden, but navigation keys (Arrows, Home, End, PgUp, PgDown) still work.
Enjoy!`

	return tui.View(
		tui.Style{
			Direction: tui.Column,
			FlexGrow:  1,
			Padding:   tui.All(2),
			Gap:       1,
		},
		tui.Text("ReadOnly TextArea Example", tui.WithTextStyle(tui.Style{Bold: true})),
		tui.TextArea(tui.TextAreaProps{
			Value:      placeholderText,
			ReadOnly:   true,
			NoWordWrap: false, // WordWrap: true by default, so NoWordWrap is false
			Style: tui.Style{
				Border: tui.BorderAll,
				Height: tui.Cells(10),
			},
			OnCopy: func(s string) {
				setCopied(s)
			},
		}),
		tui.View(
			tui.Style{
				Border:  tui.BorderAll,
				Padding: tui.All(1),
			},
			tui.Text(fmt.Sprintf("Last copied:\n%s", copied)),
		),
	)
}

func main() {
	// We want Ctrl+C to copy when dragging, but in ReadOnly it might not be strictly needed for drag
	// Let's use the standard Run
	if err := tui.Run(App); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
