package style

import (
	"testing"

	"github.com/smasonuk/earlgray/internal/color"
)

func TestMergeInheritsParentForeground(t *testing.T) {
	parent := Style{
		Foreground: color.ANSIColor(2),
	}
	child := Style{
		// foreground unspecified
	}
	result := Merge(parent, child)
	if !result.Foreground.IsSpecified() || result.Foreground.Kind != color.ANSI {
		t.Errorf("child should inherit parent foreground, got %v", result.Foreground)
	}
}

func TestMergeInheritsParentBackground(t *testing.T) {
	parent := Style{
		Background: color.ANSIColor(4),
	}
	child := Style{
		// background unspecified
	}
	result := Merge(parent, child)
	if !result.Background.IsSpecified() || result.Background.Kind != color.ANSI {
		t.Errorf("child should inherit parent background, got %v", result.Background)
	}
}

func TestMergeChildForegroundOverridesParent(t *testing.T) {
	parent := Style{
		Foreground: color.ANSIColor(2),
	}
	child := Style{
		Foreground: color.ANSIColor(3),
	}
	result := Merge(parent, child)
	if result.Foreground.ANSIVal != 3 {
		t.Errorf("child foreground should override parent, got %v", result.Foreground.ANSIVal)
	}
}

func TestMergeChildBackgroundOverridesParent(t *testing.T) {
	parent := Style{
		Background: color.ANSIColor(4),
	}
	child := Style{
		Background: color.ANSIColor(5),
	}
	result := Merge(parent, child)
	if result.Background.ANSIVal != 5 {
		t.Errorf("child background should override parent, got %v", result.Background.ANSIVal)
	}
}

func TestMergeChildDefaultColorResetsInheritance(t *testing.T) {
	parent := Style{
		Foreground: color.ANSIColor(2),
	}
	child := Style{
		Foreground: color.DefaultColor(),
	}
	result := Merge(parent, child)
	if result.Foreground.Kind != color.Default {
		t.Errorf("child default color should not inherit, got %v", result.Foreground.Kind)
	}
}

func TestMergeOtherAttributesPreserved(t *testing.T) {
	parent := Style{
		Width: Cells(10),
		Bold:  true,
	}
	child := Style{
		Italic: true,
	}
	result := Merge(parent, child)
	if result.Italic != true {
		t.Error("child italic should be preserved")
	}
	if result.Width.Value != 0 {
		t.Error("child width (unspecified) should remain unspecified")
	}
}

func TestMergeVisualInheritsAdditionalAttributes(t *testing.T) {
	parent := Style{
		Faint:         true,
		Strikethrough: true,
		Reverse:       true,
	}

	result := MergeVisual(parent, Style{})

	if !result.Faint {
		t.Error("child should inherit faint")
	}
	if !result.Strikethrough {
		t.Error("child should inherit strikethrough")
	}
	if !result.Reverse {
		t.Error("child should inherit reverse")
	}
}
