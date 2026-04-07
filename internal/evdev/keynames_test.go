package evdev

import (
	"testing"
)

func TestKeyNameAlpha(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{30, "A"}, {31, "S"}, {32, "D"}, {33, "F"},
		{44, "Z"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameNumbers(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{2, "1"}, {3, "2"}, {10, "9"}, {11, "0"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameModifiers(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{42, "Shift"}, {54, "Shift"},
		{29, "Ctrl"}, {97, "Ctrl"},
		{56, "Alt"}, {100, "Alt"},
		{125, "Super"}, {126, "Super"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameSpecial(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{1, "Esc"}, {14, "Backspace"}, {15, "Tab"},
		{28, "Enter"}, {57, "Space"},
		{111, "Delete"}, {110, "Insert"},
		{102, "Home"}, {107, "End"},
		{104, "PgDn"}, {109, "PgUp"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameArrows(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{103, "↑"}, {108, "↓"}, {105, "←"}, {106, "→"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameFunctionKeys(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{59, "F1"}, {60, "F2"}, {66, "F8"}, {67, "F9"}, {68, "F10"},
		{87, "F11"}, {88, "F12"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestKeyNameMouse(t *testing.T) {
	if got := KeyName(BtnLeft); got != "LMB" {
		t.Errorf("KeyName(BtnLeft) = %q, want LMB", got)
	}
	if got := KeyName(BtnRight); got != "RMB" {
		t.Errorf("KeyName(BtnRight) = %q, want RMB", got)
	}
}

func TestKeyNameUnknown(t *testing.T) {
	got := KeyName(0xFFFF)
	if got != "0xFFFF" {
		t.Errorf("KeyName(0xFFFF) = %q, want 0xFFFF", got)
	}
}

func TestKeyNamePunctuation(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{12, "-"}, {13, "="}, {26, "["}, {27, "]"},
		{39, ";"}, {40, "'"}, {41, "`"},
		{43, "\\"}, {51, ","}, {52, "."}, {53, "/"},
	}
	for _, tt := range tests {
		if got := KeyName(tt.code); got != tt.want {
			t.Errorf("KeyName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
