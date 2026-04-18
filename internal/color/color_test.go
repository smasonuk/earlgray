package color

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestZeroValueIsUnspecified(t *testing.T) {
	c := Color{}
	if c.IsSpecified() {
		t.Error("zero-value Color should be unspecified")
	}
	if c.Kind != Unspecified {
		t.Errorf("zero-value Kind should be Unspecified, got %v", c.Kind)
	}
}

func TestDefaultColorIsSpecified(t *testing.T) {
	c := DefaultColor()
	if !c.IsSpecified() {
		t.Error("DefaultColor should be specified")
	}
	if c.Kind != Default {
		t.Errorf("DefaultColor kind should be Default, got %v", c.Kind)
	}
}

func TestANSIColorIsSpecified(t *testing.T) {
	c := ANSIColor(2)
	if !c.IsSpecified() {
		t.Error("ANSIColor should be specified")
	}
	if c.Kind != ANSI {
		t.Errorf("ANSIColor kind should be ANSI, got %v", c.Kind)
	}
}

func TestRGBColorIsSpecified(t *testing.T) {
	c := RGBColor(255, 0, 0)
	if !c.IsSpecified() {
		t.Error("RGBColor should be specified")
	}
	if c.Kind != RGB {
		t.Errorf("RGBColor kind should be RGB, got %v", c.Kind)
	}
}

func TestToTcellUnspecified(t *testing.T) {
	c := Color{} // unspecified
	result := c.ToTcell()
	if result != tcell.ColorDefault {
		t.Errorf("unspecified color should convert to ColorDefault")
	}
}

func TestToTcellDefault(t *testing.T) {
	c := DefaultColor()
	result := c.ToTcell()
	if result != tcell.ColorDefault {
		t.Errorf("DefaultColor should convert to ColorDefault")
	}
}

func TestToTcellANSI(t *testing.T) {
	c := ANSIColor(5)
	result := c.ToTcell()
	expected := tcell.PaletteColor(5)
	if result != expected {
		t.Errorf("ANSIColor(5) should convert to PaletteColor(5)")
	}
}

func TestToTcellRGB(t *testing.T) {
	c := RGBColor(100, 150, 200)
	result := c.ToTcell()
	expected := tcell.NewRGBColor(100, 150, 200)
	if result != expected {
		t.Errorf("RGBColor should convert to NewRGBColor")
	}
}
