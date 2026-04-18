package screen

import (
	"testing"

	"github.com/smason/earlgray/internal/color"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer(10, 5)
	if b.W != 10 || b.H != 5 {
		t.Fatalf("expected 10x5, got %dx%d", b.W, b.H)
	}
	if len(b.Cells) != 50 {
		t.Fatalf("expected 50 cells, got %d", len(b.Cells))
	}
	// All cells should be spaces.
	for i, c := range b.Cells {
		if c.Rune != ' ' {
			t.Errorf("cell %d: expected space, got %q", i, c.Rune)
		}
	}
}

func TestSetAndAt(t *testing.T) {
	b := NewBuffer(5, 5)
	style := CellStyle{Fg: color.ANSIColor(2)}
	b.SetCell(2, 3, 'X', style)
	c := b.At(2, 3)
	if c.Rune != 'X' {
		t.Errorf("expected 'X', got %q", c.Rune)
	}
	if c.Style.Fg != style.Fg {
		t.Errorf("style mismatch")
	}
}

func TestSetCellOutOfBounds(t *testing.T) {
	b := NewBuffer(5, 5)
	// Should not panic.
	b.SetCell(-1, 0, 'X', CellStyle{})
	b.SetCell(5, 0, 'X', CellStyle{})
	b.SetCell(0, -1, 'X', CellStyle{})
	b.SetCell(0, 5, 'X', CellStyle{})
}

func TestClear(t *testing.T) {
	b := NewBuffer(3, 3)
	b.SetCell(1, 1, 'A', CellStyle{Bold: true})
	b.Clear()
	c := b.At(1, 1)
	if c.Rune != ' ' || c.Style.Bold {
		t.Errorf("Clear did not reset cell")
	}
}

func TestFillRect(t *testing.T) {
	b := NewBuffer(10, 10)
	style := CellStyle{Fg: color.ANSIColor(1)}
	b.FillRect(2, 2, 3, 3, '#', style)
	for y := 2; y < 5; y++ {
		for x := 2; x < 5; x++ {
			c := b.At(x, y)
			if c.Rune != '#' {
				t.Errorf("(%d,%d): expected '#', got %q", x, y, c.Rune)
			}
		}
	}
	// Corner outside rect should be space.
	if b.At(1, 1).Rune != ' ' {
		t.Error("expected space at (1,1)")
	}
}

func TestDiff(t *testing.T) {
	prev := NewBuffer(5, 5)
	next := NewBuffer(5, 5)
	next.SetCell(2, 2, 'X', CellStyle{})

	var calls []struct{ x, y int; ch rune }
	type recorder struct{}
	rec := &diffRecorder{}
	Diff(prev, next, rec)

	// Only (2,2) should be different.
	found := false
	for _, c := range rec.calls {
		if c.x == 2 && c.y == 2 && c.ch == 'X' {
			found = true
		}
	}
	if !found {
		t.Error("expected diff to find changed cell at (2,2)")
	}
	_ = calls
}

type diffCall struct{ x, y int; ch rune }
type diffRecorder struct{ calls []diffCall }
func (r *diffRecorder) SetCell(x, y int, ch rune, style CellStyle) {
	r.calls = append(r.calls, diffCall{x, y, ch})
}

func TestDiffNilPrev(t *testing.T) {
	next := NewBuffer(2, 2)
	rec := &diffRecorder{}
	Diff(nil, next, rec)
	if len(rec.calls) != 4 {
		t.Errorf("expected 4 diff calls with nil prev, got %d", len(rec.calls))
	}
}

func TestDrawHLine(t *testing.T) {
	b := NewBuffer(10, 5)
	b.DrawHLine(1, 2, 5, '-', CellStyle{})
	for x := 1; x < 6; x++ {
		if b.At(x, 2).Rune != '-' {
			t.Errorf("expected '-' at (%d, 2)", x)
		}
	}
	if b.At(0, 2).Rune != ' ' {
		t.Error("expected space at (0,2)")
	}
}

func TestDrawVLine(t *testing.T) {
	b := NewBuffer(5, 10)
	b.DrawVLine(3, 1, 5, '|', CellStyle{})
	for y := 1; y < 6; y++ {
		if b.At(3, y).Rune != '|' {
			t.Errorf("expected '|' at (3, %d)", y)
		}
	}
	if b.At(3, 0).Rune != ' ' {
		t.Error("expected space at (3,0)")
	}
}

func TestDrawTextClipped(t *testing.T) {
	b := NewBuffer(10, 5)
	b.DrawTextClipped(0, 0, "Hello, World!", CellStyle{}, 0, 0, 5, 1)
	// Only "Hello" should fit.
	want := "Hello"
	for i, ch := range want {
		if b.At(i, 0).Rune != ch {
			t.Errorf("pos %d: expected %q, got %q", i, ch, b.At(i, 0).Rune)
		}
	}
	// Position 5 should still be space.
	if b.At(5, 0).Rune != ' ' {
		t.Errorf("expected space at (5,0), got %q", b.At(5, 0).Rune)
	}
}

func TestDrawWideCharacter(t *testing.T) {
	// "界" is a CJK character with width 2
	b := NewBuffer(5, 1)
	b.DrawTextClipped(0, 0, "a界c", CellStyle{}, 0, 0, 5, 1)
	// First cell should be 'a'
	if b.At(0, 0).Rune != 'a' {
		t.Errorf("pos 0: expected 'a', got %q", b.At(0, 0).Rune)
	}
	// Second cell should be '界'
	if b.At(1, 0).Rune != '界' {
		t.Errorf("pos 1: expected '界', got %q", b.At(1, 0).Rune)
	}
	// Third cell should be a space (filler for wide char)
	if b.At(2, 0).Rune != ' ' {
		t.Errorf("pos 2: expected space, got %q", b.At(2, 0).Rune)
	}
	// Fourth cell should be 'c'
	if b.At(3, 0).Rune != 'c' {
		t.Errorf("pos 3: expected 'c', got %q", b.At(3, 0).Rune)
	}
}

func TestDrawWideCharacterNotPartiallyClipped(t *testing.T) {
	// "界" is width 2, so if clip boundary is at column 2, it shouldn't be drawn
	b := NewBuffer(10, 1)
	// Draw "a界" with clip boundary at column 2 (only room for 1 wide char)
	// "a" takes column 0, but "界" would need columns 1-2, and clip is at 2
	// So "界" should not be drawn at all
	b.DrawTextClipped(0, 0, "a界", CellStyle{}, 0, 0, 2, 1)
	// Position 0 should have 'a'
	if b.At(0, 0).Rune != 'a' {
		t.Errorf("pos 0: expected 'a', got %q", b.At(0, 0).Rune)
	}
	// Position 1 should be space (nothing written there)
	if b.At(1, 0).Rune != ' ' {
		t.Errorf("pos 1: expected space, got %q", b.At(1, 0).Rune)
	}
}
