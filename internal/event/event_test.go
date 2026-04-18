package event

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/smason/earlgray/internal/input"
)

func TestNormalizeKeyRune(t *testing.T) {
	tests := []struct {
		name     string
		tcellKey tcell.Key
		rune     rune
		want     input.Key
	}{
		{
			name:     "printable rune",
			tcellKey: tcell.KeyRune,
			rune:     'a',
			want:     input.KeyRune,
		},
		{
			name:     "space rune",
			tcellKey: tcell.KeyRune,
			rune:     ' ',
			want:     input.KeyRune,
		},
		{
			name:     "number rune",
			tcellKey: tcell.KeyRune,
			rune:     '5',
			want:     input.KeyRune,
		},
		{
			name:     "empty rune",
			tcellKey: tcell.KeyRune,
			rune:     0,
			want:     input.KeyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKey(tt.tcellKey, tt.rune)
			if got != tt.want {
				t.Errorf("NormalizeKey(%v, %q) = %v, want %v", tt.tcellKey, tt.rune, got, tt.want)
			}
		})
	}
}

func TestNormalizeKeySpecialKeys(t *testing.T) {
	tests := []struct {
		name     string
		tcellKey tcell.Key
		want     input.Key
	}{
		{"Enter", tcell.KeyEnter, input.KeyEnter},
		{"Escape", tcell.KeyEsc, input.KeyEsc},
		{"Tab", tcell.KeyTab, input.KeyTab},
		{"Up", tcell.KeyUp, input.KeyUp},
		{"Down", tcell.KeyDown, input.KeyDown},
		{"Left", tcell.KeyLeft, input.KeyLeft},
		{"Right", tcell.KeyRight, input.KeyRight},
		{"Home", tcell.KeyHome, input.KeyHome},
		{"End", tcell.KeyEnd, input.KeyEnd},
		{"PgUp", tcell.KeyPgUp, input.KeyPgUp},
		{"PgDn", tcell.KeyPgDn, input.KeyPgDown},
		{"Delete", tcell.KeyDelete, input.KeyDelete},
		{"Insert", tcell.KeyInsert, input.KeyInsert},
		{"Backspace", tcell.KeyBackspace, input.KeyBackspace},
		{"Backspace2", tcell.KeyBackspace2, input.KeyBackspace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKey(tt.tcellKey, 0)
			if got != tt.want {
				t.Errorf("NormalizeKey(%v, 0) = %v, want %v", tt.tcellKey, got, tt.want)
			}
		})
	}
}

func TestNormalizeKeyUnknown(t *testing.T) {
	// Some arbitrary key code that we don't handle
	unknownKey := tcell.Key(9999)
	got := NormalizeKey(unknownKey, 0)
	if got != input.KeyUnknown {
		t.Errorf("NormalizeKey(unknown) = %v, want KeyUnknown", got)
	}
}

func TestNormalizeKeyRuneWithNonRuneKey(t *testing.T) {
	// If we have a rune but tcellKey is not KeyRune, still return KeyRune
	got := NormalizeKey(tcell.KeyEnter, 'x')
	if got != input.KeyRune {
		t.Errorf("NormalizeKey(KeyEnter, 'x') = %v, want KeyRune", got)
	}
}

func TestKeyIsCtrlCKeyCtrlC(t *testing.T) {
	k := Key{Key: tcell.KeyCtrlC}
	if !k.IsCtrlC() {
		t.Error("KeyCtrlC should be recognized as Ctrl-C")
	}
}

func TestKeyIsCtrlCModifiedRuneLowercase(t *testing.T) {
	k := Key{Key: tcell.KeyRune, Rune: 'c', Mod: tcell.ModCtrl}
	if !k.IsCtrlC() {
		t.Error("Ctrl-c rune should be recognized as Ctrl-C")
	}
}

func TestKeyIsCtrlCModifiedRuneUppercase(t *testing.T) {
	k := Key{Key: tcell.KeyRune, Rune: 'C', Mod: tcell.ModCtrl}
	if !k.IsCtrlC() {
		t.Error("Ctrl-C rune should be recognized as Ctrl-C")
	}
}

func TestKeyIsCtrlCPlainCIsFalse(t *testing.T) {
	k := Key{Key: tcell.KeyRune, Rune: 'c'}
	if k.IsCtrlC() {
		t.Error("plain c should not be recognized as Ctrl-C")
	}
}

func TestKeyIsTabPlainTab(t *testing.T) {
	k := Key{Key: tcell.KeyTab}
	if !k.IsTab() {
		t.Error("KeyTab should be recognized as Tab")
	}
}

func TestKeyIsTabWithShiftIsFalse(t *testing.T) {
	k := Key{Key: tcell.KeyTab, Mod: tcell.ModShift}
	if k.IsTab() {
		t.Error("Shift+Tab should not be recognized as plain Tab")
	}
}

func TestKeyIsShiftTabBacktab(t *testing.T) {
	k := Key{Key: tcell.KeyBacktab}
	if !k.IsShiftTab() {
		t.Error("KeyBacktab should be recognized as Shift+Tab")
	}
}

func TestKeyIsShiftTabTabWithShift(t *testing.T) {
	k := Key{Key: tcell.KeyTab, Mod: tcell.ModShift}
	if !k.IsShiftTab() {
		t.Error("Tab with Shift modifier should be recognized as Shift+Tab")
	}
}

func TestKeyIsShiftTabPlainTabIsFalse(t *testing.T) {
	k := Key{Key: tcell.KeyTab}
	if k.IsShiftTab() {
		t.Error("plain Tab should not be recognized as Shift+Tab")
	}
}
