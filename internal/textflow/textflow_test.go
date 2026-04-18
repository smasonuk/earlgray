package textflow

import (
	"reflect"
	"testing"
)

func TestVisualLines(t *testing.T) {
	tests := []struct {
		text     string
		wordWrap bool
		width    int
		want     []string
	}{
		{"hello", true, 0, []string{"hello"}},
		{"hello world", true, 5, []string{"hello", "world"}},
		{"hello world", false, 5, []string{"hello world"}},
		{"a b c", true, 1, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		got := VisualLines(tt.text, tt.wordWrap, tt.width)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("VisualLines(%q, %v, %d) = %q, want %q", tt.text, tt.wordWrap, tt.width, got, tt.want)
		}
	}
}

func TestMaxLineWidth(t *testing.T) {
	tests := []struct {
		lines []string
		want  int
	}{
		{[]string{"a界b"}, 4},
		{[]string{"hello", "world!"}, 6},
		{[]string{"", "abc"}, 3},
	}

	for _, tt := range tests {
		got := MaxLineWidth(tt.lines)
		if got != tt.want {
			t.Errorf("MaxLineWidth(%q) = %d, want %d", tt.lines, got, tt.want)
		}
	}
}

func TestWrapLines(t *testing.T) {
	tests := []struct {
		text  string
		width int
		want  []string
	}{
		{"alpha beta gamma", 10, []string{"alpha beta", "gamma"}},
		{"世界abc", 3, []string{"世", "界a", "bc"}},
		{"verylongword", 5, []string{"veryl", "ongwo", "rd"}},
	}

	for _, tt := range tests {
		got := WrapLines(tt.text, tt.width)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("WrapLines(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
		}
	}
}

func TestVisualLinesNoWrapSplitsNewlines(t *testing.T) {
	got := VisualLines("one\ntwo\nthree", false, 10)
	want := []string{"one", "two", "three"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestVisualLinesWrapWithZeroWidthFallsBackToLogicalLines(t *testing.T) {
	got := VisualLines("hello\nworld", true, 0)
	want := []string{"hello", "world"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWrapLinesWrapsAtWordBoundary(t *testing.T) {
	got := WrapLines("alpha beta gamma", 10)
	want := []string{"alpha beta", "gamma"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWrapLinesHardBreaksLongWord(t *testing.T) {
	got := WrapLines("abcdef", 3)
	want := []string{"abc", "def"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWrapLinesHonorsWideRuneWidth(t *testing.T) {
	got := WrapLines("界abc", 3)
	want := []string{"界a", "bc"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMaxLineWidthHonorsWideRunes(t *testing.T) {
	got := MaxLineWidth([]string{"abc", "a界b", "x"})
	if got != 4 {
		t.Fatalf("MaxLineWidth = %d, want 4", got)
	}
}

func TestWrapLinesPreservesBlankLines(t *testing.T) {
	got := WrapLines("one\n\nthree", 10)
	want := []string{"one", "", "three"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestVisualLinesEmptyTextIsOneEmptyLine(t *testing.T) {
	got := VisualLines("", false, 10)
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("VisualLines empty = %#v, want []string{\"\"}", got)
	}

	got = VisualLines("", true, 10)
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("VisualLines empty wrapped = %#v, want []string{\"\"}", got)
	}
}

func TestWrapLinesCurrentWhitespaceBehavior(t *testing.T) {
	got := WrapLines("alpha  beta", 20)
	want := []string{"alpha  beta"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
