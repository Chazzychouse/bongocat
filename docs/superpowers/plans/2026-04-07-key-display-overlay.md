# Key Display Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the sprite-based bongocat overlay with a Cairo text renderer that displays the actual key/mouse button pressed, with a fading trail of recent inputs.

**Architecture:** Same evdev → state → Cairo draw pipeline. State machine becomes a ring buffer of recent key names. Renderer swaps sprite blitting for `ShowText`/path drawing. No new dependencies.

**Tech Stack:** Go, GTK4 (gotk4 bindings), Cairo text API, Linux evdev

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/evdev/keynames.go` | `KeyName(code uint16) string` mapping |
| Create | `internal/evdev/keynames_test.go` | Tests for key name mapping |
| Rewrite | `cmd/overlay/state.go` | Ring buffer of recent key entries |
| Rewrite | `cmd/overlay/state_test.go` | Tests for ring buffer state |
| Rewrite | `cmd/overlay/render.go` | Cairo text + mouse icon rendering |
| Modify | `cmd/overlay/main.go` | Wire new state/renderer, remove manifest |
| Delete | `cmd/overlay/manifest.go` | No longer needed |
| Delete | `cmd/overlay/manifest_test.go` | No longer needed |

---

### Task 1: Key Name Mapping

**Files:**
- Create: `internal/evdev/keynames.go`
- Create: `internal/evdev/keynames_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/evdev/keynames_test.go`:

```go
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
		{42, "Shift"}, {54, "Shift"},   // left and right shift
		{29, "Ctrl"}, {97, "Ctrl"},     // left and right ctrl
		{56, "Alt"}, {100, "Alt"},      // left and right alt
		{125, "Super"}, {126, "Super"}, // left and right super
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/chazzy/projects/bongocat && go test ./internal/evdev/ -v -run TestKeyName`
Expected: compilation error — `KeyName` not defined

- [ ] **Step 3: Implement KeyName**

Create `internal/evdev/keynames.go`:

```go
package evdev

import "fmt"

// keyNames maps Linux evdev key codes to human-readable display labels.
// Codes from <linux/input-event-codes.h>.
var keyNames = map[uint16]string{
	// Row 1: Esc + number row
	1: "Esc", 2: "1", 3: "2", 4: "3", 5: "4", 6: "5",
	7: "6", 8: "7", 9: "8", 10: "9", 11: "0", 12: "-", 13: "=",
	14: "Backspace",

	// Row 2: Tab + QWERTY
	15: "Tab",
	16: "Q", 17: "W", 18: "E", 19: "R", 20: "T", 21: "Y",
	22: "U", 23: "I", 24: "O", 25: "P", 26: "[", 27: "]",
	28: "Enter",

	// Row 3: Caps + ASDF
	29: "Ctrl", // left ctrl
	30: "A", 31: "S", 32: "D", 33: "F", 34: "G", 35: "H",
	36: "J", 37: "K", 38: "L", 39: ";", 40: "'", 41: "`",

	// Row 4: Shift + ZXCV
	42: "Shift", // left shift
	43: "\\",
	44: "Z", 45: "X", 46: "C", 47: "V", 48: "B", 49: "N",
	50: "M", 51: ",", 52: ".", 53: "/",
	54: "Shift", // right shift

	// Bottom row
	56:  "Alt",   // left alt
	57:  "Space",
	58:  "Caps",
	100: "Alt",   // right alt
	97:  "Ctrl",  // right ctrl
	125: "Super", // left super
	126: "Super", // right super

	// Function keys
	59: "F1", 60: "F2", 61: "F3", 62: "F4", 63: "F5", 64: "F6",
	65: "F7", 66: "F8", 67: "F9", 68: "F10", 87: "F11", 88: "F12",

	// Navigation
	102: "Home", 107: "End", 104: "PgDn", 109: "PgUp",
	110: "Insert", 111: "Delete",

	// Arrow keys
	103: "↑", 108: "↓", 105: "←", 106: "→",

	// Mouse buttons
	BtnLeft:  "LMB",
	BtnRight: "RMB",
}

// KeyName returns the display label for an evdev key code.
// Unknown codes are formatted as hex (e.g. "0x1F").
func KeyName(code uint16) string {
	if name, ok := keyNames[code]; ok {
		return name
	}
	return fmt.Sprintf("0x%X", code)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/chazzy/projects/bongocat && go test ./internal/evdev/ -v -run TestKeyName`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/evdev/keynames.go internal/evdev/keynames_test.go
git commit -m "feat: add evdev key code to display name mapping"
```

---

### Task 2: Ring Buffer State Machine

**Files:**
- Rewrite: `cmd/overlay/state.go`
- Rewrite: `cmd/overlay/state_test.go`

- [ ] **Step 1: Write the failing tests**

Rewrite `cmd/overlay/state_test.go`:

```go
package main

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := newState()
	if s.active() {
		t.Error("new state should not be active")
	}
	if len(s.entries()) != 0 {
		t.Errorf("entries = %d, want 0", len(s.entries()))
	}
}

func TestPushAddsEntries(t *testing.T) {
	s := newState()
	s.push("A", false)
	entries := s.entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].name != "A" {
		t.Errorf("name = %q, want A", entries[0].name)
	}
	if entries[0].isMouse {
		t.Error("isMouse should be false")
	}
}

func TestPushNewestFirst(t *testing.T) {
	s := newState()
	s.push("A", false)
	s.push("B", false)
	s.push("C", false)
	entries := s.entries()
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if entries[0].name != "C" {
		t.Errorf("entries[0] = %q, want C", entries[0].name)
	}
	if entries[1].name != "B" {
		t.Errorf("entries[1] = %q, want B", entries[1].name)
	}
	if entries[2].name != "A" {
		t.Errorf("entries[2] = %q, want A", entries[2].name)
	}
}

func TestPushCapsAtFive(t *testing.T) {
	s := newState()
	for _, k := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		s.push(k, false)
	}
	entries := s.entries()
	if len(entries) != 5 {
		t.Fatalf("entries = %d, want 5", len(entries))
	}
	if entries[0].name != "G" {
		t.Errorf("newest = %q, want G", entries[0].name)
	}
	if entries[4].name != "C" {
		t.Errorf("oldest = %q, want C", entries[4].name)
	}
}

func TestPushMouse(t *testing.T) {
	s := newState()
	s.push("LMB", true)
	entries := s.entries()
	if !entries[0].isMouse {
		t.Error("isMouse should be true")
	}
}

func TestActive(t *testing.T) {
	s := newState()
	if s.active() {
		t.Error("should not be active when empty")
	}
	s.push("A", false)
	if !s.active() {
		t.Error("should be active after push")
	}
}

func TestTickIdleTimeout(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond
	s.push("A", false)

	time.Sleep(100 * time.Millisecond)
	changed := s.tick()
	if !changed {
		t.Error("tick should return true when clearing")
	}
	if s.active() {
		t.Error("should be idle after timeout")
	}
	if len(s.entries()) != 0 {
		t.Errorf("entries = %d, want 0", len(s.entries()))
	}
}

func TestTickNoChangeWhileActive(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond
	s.push("A", false)

	time.Sleep(30 * time.Millisecond)
	s.push("B", false)
	time.Sleep(30 * time.Millisecond)
	changed := s.tick()
	if changed {
		t.Error("tick should return false — input was recent")
	}
	if !s.active() {
		t.Error("should still be active")
	}
}

func TestTickNoChangeWhenAlreadyIdle(t *testing.T) {
	s := newState()
	changed := s.tick()
	if changed {
		t.Error("tick on empty state should return false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/chazzy/projects/bongocat && go test ./cmd/overlay/ -v -run "TestNewState|TestPush|TestActive|TestTick" 2>&1 | head -30`
Expected: compilation errors — `active`, `entries`, `push` not defined

- [ ] **Step 3: Implement ring buffer state**

Rewrite `cmd/overlay/state.go`:

```go
package main

import "time"

const maxHistory = 5

type keyEntry struct {
	name    string
	isMouse bool
	time    time.Time
}

type state struct {
	history     []keyEntry
	lastInput   time.Time
	idleTimeout time.Duration
}

func newState() *state {
	return &state{
		idleTimeout: 200 * time.Millisecond,
	}
}

func (s *state) push(name string, isMouse bool) {
	s.lastInput = time.Now()
	entry := keyEntry{name: name, isMouse: isMouse, time: s.lastInput}
	s.history = append([]keyEntry{entry}, s.history...)
	if len(s.history) > maxHistory {
		s.history = s.history[:maxHistory]
	}
}

func (s *state) tick() bool {
	if len(s.history) == 0 {
		return false
	}
	if time.Since(s.lastInput) >= s.idleTimeout {
		s.history = s.history[:0]
		return true
	}
	return false
}

func (s *state) entries() []keyEntry {
	return s.history
}

func (s *state) active() bool {
	return len(s.history) > 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/chazzy/projects/bongocat && go test ./cmd/overlay/ -v -run "TestNewState|TestPush|TestActive|TestTick"`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/overlay/state.go cmd/overlay/state_test.go
git commit -m "feat: replace paw state machine with key history ring buffer"
```

---

### Task 3: Cairo Text Renderer

**Files:**
- Rewrite: `cmd/overlay/render.go`

- [ ] **Step 1: Rewrite the renderer**

Rewrite `cmd/overlay/render.go`:

```go
package main

import (
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
)

type renderer struct{}

func newRenderer() *renderer {
	return &renderer{}
}

func (r *renderer) draw(cr *cairo.Context, entries []keyEntry, w, h int) {
	if len(entries) == 0 {
		return
	}

	cr.SelectFontFace("monospace", cairo.FontSlantNormal, cairo.FontWeightBold)

	fontSize := float64(h) * 0.5
	opacity := 1.0
	y := float64(h) * 0.45 // start near vertical center

	for i, entry := range entries {
		if entry.isMouse {
			r.drawMouse(cr, entry.name, float64(w), y, fontSize, opacity)
		} else {
			r.drawKey(cr, entry.name, float64(w), y, fontSize, opacity)
		}

		// Trail: each subsequent key is smaller and more transparent
		if i == 0 {
			y += fontSize * 0.7
		} else {
			y += fontSize * 0.6
		}
		fontSize *= 0.7
		opacity *= 0.6

		if y > float64(h) {
			break
		}
	}
}

func (r *renderer) drawKey(cr *cairo.Context, name string, w, y, fontSize, opacity float64) {
	cr.SetFontSize(fontSize)
	ext := cr.TextExtents(name)
	x := (w - ext.Width) / 2 - ext.XBearing
	cr.MoveTo(x, y)
	cr.SetSourceRGBA(1, 1, 1, opacity)
	cr.ShowText(name)
}

func (r *renderer) drawMouse(cr *cairo.Context, name string, w, y, fontSize, opacity float64) {
	// Mouse icon dimensions scaled to match text size
	mw := fontSize * 0.5  // mouse width
	mh := fontSize * 0.75 // mouse height
	x := (w - mw) / 2
	top := y - mh*0.7
	radius := mw * 0.25

	cr.SetLineWidth(2)

	// Body outline: rounded rectangle
	cr.NewPath()
	cr.Arc(x+mw-radius, top+radius, radius, -math.Pi/2, 0)
	cr.Arc(x+mw-radius, top+mh-radius, radius, 0, math.Pi/2)
	cr.Arc(x+radius, top+mh-radius, radius, math.Pi/2, math.Pi)
	cr.Arc(x+radius, top+radius, radius, math.Pi, 3*math.Pi/2)
	cr.ClosePath()
	cr.SetSourceRGBA(1, 1, 1, opacity*0.4)
	cr.FillPreserve()
	cr.SetSourceRGBA(1, 1, 1, opacity)
	cr.Stroke()

	// Divider line at top half
	mid := x + mw/2
	split := top + mh*0.45
	cr.MoveTo(mid, top+radius*0.5)
	cr.LineTo(mid, split)
	cr.SetSourceRGBA(1, 1, 1, opacity)
	cr.Stroke()

	// Horizontal split between buttons and body
	cr.MoveTo(x+radius*0.5, split)
	cr.LineTo(x+mw-radius*0.5, split)
	cr.Stroke()

	// Fill the clicked button
	if name == "LMB" {
		// Fill left button region
		cr.Rectangle(x+radius*0.5, top+radius*0.5, mw/2-radius*0.5, split-top-radius*0.5)
		cr.SetSourceRGBA(1, 1, 1, opacity*0.8)
		cr.Fill()
	} else if name == "RMB" {
		// Fill right button region
		cr.Rectangle(mid, top+radius*0.5, mw/2-radius*0.5, split-top-radius*0.5)
		cr.SetSourceRGBA(1, 1, 1, opacity*0.8)
		cr.Fill()
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/chazzy/projects/bongocat && go build ./cmd/overlay/ 2>&1 | head -20`
Expected: compilation errors from `main.go` (still references old types) — that's expected and will be fixed in Task 4. The renderer itself should have no syntax errors. To isolate, check with `go vet`:

Run: `cd /home/chazzy/projects/bongocat && go vet ./cmd/overlay/render.go 2>&1 || true`

- [ ] **Step 3: Commit**

```bash
git add cmd/overlay/render.go
git commit -m "feat: replace sprite renderer with Cairo text and mouse icon drawing"
```

---

### Task 4: Wire Up Main Loop and Delete Manifest

**Files:**
- Modify: `cmd/overlay/main.go`
- Delete: `cmd/overlay/manifest.go`
- Delete: `cmd/overlay/manifest_test.go`

- [ ] **Step 1: Rewrite main.go**

Replace `cmd/overlay/main.go` with:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/chazzy/bongocat/internal/evdev"
)

func main() {
	log.SetFlags(0)

	winSize := flag.Int("size", 200, "window size in pixels")
	flag.Parse()

	keyboards, mice := evdev.DetectDevices()
	if len(keyboards) == 0 && len(mice) == 0 {
		log.Fatal("No input devices detected")
	}
	for _, d := range keyboards {
		log.Printf("keyboard: %s (%s)", d.Name, d.Path)
	}
	for _, d := range mice {
		log.Printf("mouse:    %s (%s)", d.Name, d.Path)
	}

	events := make(chan evdev.KeyEvent, 256)
	var devices []*os.File
	for _, d := range slices.Concat(keyboards, mice) {
		f, err := evdev.OpenDevice(d.Path)
		if err != nil {
			log.Printf("warning: cannot open %s: %v", d.Path, err)
			continue
		}
		devices = append(devices, f)
		go evdev.ReadDevice(f, events)
	}
	defer func() {
		for _, f := range devices {
			f.Close()
		}
	}()

	if len(devices) == 0 {
		log.Fatal("No devices could be opened")
	}

	app := gtk.NewApplication("com.github.bongocat", gio.ApplicationFlagsNone)

	app.ConnectActivate(func() {
		win := gtk.NewApplicationWindow(app)
		win.SetTitle("bongocat")
		win.SetDefaultSize(*winSize, *winSize)
		win.SetDecorated(false)

		css := gtk.NewCSSProvider()
		css.LoadFromString("window, drawingarea { background-color: transparent; }")
		gtk.StyleContextAddProviderForDisplay(
			gdk.DisplayGetDefault(),
			css,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)

		rend := newRenderer()
		st := newState()

		area := gtk.NewDrawingArea()
		area.SetContentWidth(*winSize)
		area.SetContentHeight(*winSize)
		area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
			rend.draw(cr, st.entries(), w, h)
		})

		win.SetChild(area)

		// SIGUSR2 toggle visibility
		visible := true
		sigToggle := make(chan os.Signal, 1)
		signal.Notify(sigToggle, syscall.SIGUSR2)
		go func() {
			for range sigToggle {
				glib.IdleAdd(func() {
					visible = !visible
					win.SetVisible(visible)
				})
			}
		}()

		// Drain evdev events into state machine on GTK thread
		go func() {
			for ev := range events {
				glib.IdleAdd(func() {
					name := evdev.KeyName(ev.Code)
					isMouse := ev.Code == evdev.BtnLeft || ev.Code == evdev.BtnRight
					st.push(name, isMouse)
					area.QueueDraw()
				})
			}
		}()

		// Idle check timer (~60 ticks/sec)
		glib.TimeoutAdd(16, func() bool {
			if st.tick() {
				area.QueueDraw()
			}
			return true
		})

		win.Show()
		fmt.Printf("overlay: %dx%d\n", *winSize, *winSize)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
```

- [ ] **Step 2: Delete manifest files**

```bash
rm cmd/overlay/manifest.go cmd/overlay/manifest_test.go
```

- [ ] **Step 3: Verify build succeeds**

Run: `cd /home/chazzy/projects/bongocat && go build ./cmd/overlay/`
Expected: clean build, no errors

- [ ] **Step 4: Run all tests**

Run: `cd /home/chazzy/projects/bongocat && go test ./... -v`
Expected: all tests pass (evdev keyname tests + state ring buffer tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/overlay/main.go
git rm cmd/overlay/manifest.go cmd/overlay/manifest_test.go
git commit -m "feat: wire key display overlay, remove sprite/manifest system"
```

---

### Task 5: Manual Smoke Test

- [ ] **Step 1: Build the binary**

Run: `cd /home/chazzy/projects/bongocat && go build -o overlay ./cmd/overlay/`
Expected: binary created at `./overlay`

- [ ] **Step 2: Run and verify key display**

Run: `./overlay -size 200`

Verify:
- Window appears as transparent overlay
- Pressing keyboard keys shows the key name centered in the window
- Older keys trail below, getting smaller and more transparent
- Mouse clicks show a mouse icon with the correct button highlighted
- After 200ms of no input, display clears
- SIGUSR2 toggles visibility: `kill -USR2 $(pgrep -f './overlay')`

- [ ] **Step 3: Commit any fixes from smoke test**

If adjustments needed (font size, spacing, opacity), fix and commit.
