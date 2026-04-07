package evdev

import "fmt"

var keyNames = map[uint16]string{
	1: "Esc", 2: "1", 3: "2", 4: "3", 5: "4", 6: "5",
	7: "6", 8: "7", 9: "8", 10: "9", 11: "0", 12: "-", 13: "=",
	14: "Backspace",
	15: "Tab",
	16: "Q", 17: "W", 18: "E", 19: "R", 20: "T", 21: "Y",
	22: "U", 23: "I", 24: "O", 25: "P", 26: "[", 27: "]",
	28: "Enter",
	29: "Ctrl",
	30: "A", 31: "S", 32: "D", 33: "F", 34: "G", 35: "H",
	36: "J", 37: "K", 38: "L", 39: ";", 40: "'", 41: "`",
	42: "Shift",
	43: "\\",
	44: "Z", 45: "X", 46: "C", 47: "V", 48: "B", 49: "N",
	50: "M", 51: ",", 52: ".", 53: "/",
	54:  "Shift",
	56:  "Alt",
	57:  "Space",
	58:  "Caps",
	97:  "Ctrl",
	100: "Alt",
	125: "Super",
	126: "Super",
	59: "F1", 60: "F2", 61: "F3", 62: "F4", 63: "F5", 64: "F6",
	65: "F7", 66: "F8", 67: "F9", 68: "F10", 87: "F11", 88: "F12",
	102: "Home", 107: "End", 104: "PgDn", 109: "PgUp",
	110: "Insert", 111: "Delete",
	103: "↑", 108: "↓", 105: "←", 106: "→",
	BtnLeft:  "LMB",
	BtnRight: "RMB",
}

func KeyName(code uint16) string {
	if name, ok := keyNames[code]; ok {
		return name
	}
	return fmt.Sprintf("0x%X", code)
}
