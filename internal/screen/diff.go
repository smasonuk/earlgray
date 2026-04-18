package screen

// Differ is a function that receives changed cells during a diff.
type Differ interface {
	SetCell(x, y int, ch rune, style CellStyle)
}

// Diff compares two buffers and calls d.SetCell for each changed cell.
// prev may be nil, in which case all cells are considered changed.
func Diff(prev, next *Buffer, d Differ) {
	for y := 0; y < next.H; y++ {
		for x := 0; x < next.W; x++ {
			c := next.At(x, y)
			if prev != nil && x < prev.W && y < prev.H {
				p := prev.At(x, y)
				if p == c {
					continue
				}
			}
			d.SetCell(x, y, c.Rune, c.Style)
		}
	}
}
