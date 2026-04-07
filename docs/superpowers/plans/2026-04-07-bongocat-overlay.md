# Bongocat Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GTK4 overlay window that animates a bongo cat in response to keyboard and mouse input.

**Architecture:** Second binary (`bongocat-overlay`) in `cmd/overlay/`, sharing evdev input code via `internal/evdev/`. Uses gotk4 for GTK4 rendering with Hyprland window rules for floating/pinning. Sprite sheets with JSON manifests for swappable animations.

**Tech Stack:** Go 1.26, gotk4 (GTK4 bindings), Cairo, GdkPixbuf, evdev

---

## File Structure

```
bongocat/
  internal/
    evdev/
      evdev.go              # extracted from root evdev.go, exported API
  cmd/
    waybar/
      main.go               # current root main.go, moved here
    overlay/
      main.go               # GTK4 app, signal handling, event loop
      render.go             # sprite sheet loading, frame cropping, drawing
      state.go              # animation state machine
      state_test.go         # state machine tests
      manifest.go           # manifest JSON loading
      manifest_test.go      # manifest tests
  animations/
    bongocat/
      manifest.json         # default animation manifest
      sprite.png            # default sprite sheet (user-provided)
  config/
    bongocat-overlay.service  # systemd unit
    hyprland.conf             # example window rules
    bongocat-waybar.service   # existing
    waybar.jsonc              # existing
    waybar.css                # existing
  install.sh                  # updated for both binaries
```

---

### Task 1: Extract evdev into shared package

Move evdev code into `internal/evdev/` with exported names so both binaries can import it.

**Files:**
- Create: `internal/evdev/evdev.go`
- Delete: `evdev.go` (root)
- Modify: `cmd/waybar/main.go` (moved from `main.go`)

- [ ] **Step 1: Create `internal/evdev/evdev.go`**

Copy the contents of root `evdev.go`, change package to `evdev`, and export all types and functions. The constants stay unexported since they're only used internally within the package — except `BtnLeft` and `BtnRight` which the consumers need for distinguishing click types.

```go
package evdev

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unsafe"
)

const (
	evKey   = 0x01
	evRel   = 0x02
	keyDown = 1

	keyA = 30
	relX = 0x00
	relY = 0x01

	BtnLeft  = 0x110
	BtnRight = 0x111
)

const inputEventSize = 24

func evIoc(dir, typ, nr, size uintptr) uintptr {
	return (dir << 30) | (size << 16) | (typ << 8) | nr
}

func eviocgname(length uintptr) uintptr { return evIoc(2, 'E', 0x06, length) }
func eviocgbit(ev, length uintptr) uintptr { return evIoc(2, 'E', 0x20+ev, length) }

func evdevName(fd uintptr) string {
	buf := make([]byte, 256)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgname(256), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func evdevBits(fd uintptr, evType, maxBit int) []byte {
	size := (maxBit + 7) / 8
	bits := make([]byte, size)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgbit(uintptr(evType), uintptr(size)), uintptr(unsafe.Pointer(&bits[0])))
	if errno != 0 {
		return nil
	}
	return bits
}

func hasBit(bits []byte, bit int) bool {
	idx := bit / 8
	if idx >= len(bits) {
		return false
	}
	return bits[idx]&(1<<uint(bit%8)) != 0
}

type InputDevice struct {
	Path string
	Name string
}

func probeDevice(path string) (isKb, isMouse bool, name string) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false, false, ""
	}
	defer f.Close()

	fd := f.Fd()
	name = evdevName(fd)

	evBits := evdevBits(fd, 0, 32)
	if evBits == nil {
		return false, false, name
	}

	if hasBit(evBits, evKey) {
		if keyBits := evdevBits(fd, evKey, 768); keyBits != nil {
			isKb = hasBit(keyBits, keyA)
		}
	}

	if hasBit(evBits, evRel) {
		relBits := evdevBits(fd, evRel, 64)
		keyBits := evdevBits(fd, evKey, 768)
		if relBits != nil && keyBits != nil &&
			hasBit(relBits, relX) && hasBit(relBits, relY) &&
			hasBit(keyBits, BtnLeft) {
			isMouse = true
		}
	}

	return isKb, isMouse, name
}

func DetectDevices() (keyboards, mice []InputDevice) {
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, path := range paths {
		isKb, isMouse, name := probeDevice(path)
		dev := InputDevice{Path: path, Name: name}
		if isKb {
			keyboards = append(keyboards, dev)
		}
		if isMouse {
			mice = append(mice, dev)
		}
	}

	if makima := slices.DeleteFunc(slices.Clone(keyboards), func(d InputDevice) bool {
		return !strings.Contains(strings.ToLower(d.Name), "makima")
	}); len(makima) > 0 {
		keyboards = makima
	}

	return
}

type KeyEvent struct {
	Code uint16
}

func OpenDevice(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

func ReadDevice(f *os.File, ch chan<- KeyEvent) {
	buf := make([]byte, inputEventSize*64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return
		}
		for off := 0; off+inputEventSize <= n; off += inputEventSize {
			typ := binary.LittleEndian.Uint16(buf[off+16:])
			code := binary.LittleEndian.Uint16(buf[off+18:])
			value := int32(binary.LittleEndian.Uint32(buf[off+20:]))

			if typ == evKey && value == keyDown {
				ch <- KeyEvent{Code: code}
			}
		}
	}
}
```

- [ ] **Step 2: Move `main.go` to `cmd/waybar/main.go`**

Move the file and update it to import the shared evdev package. Replace all references to the old local types/functions with the exported names from `internal/evdev`.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/chazzy/bongocat/internal/evdev"
)

const (
	saveInterval = 10 * time.Second
	waybarSignal = 10
)

var dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "bongocat")

func stateFile() string    { return filepath.Join(dataDir, "state.json") }
func lifetimeFile() string { return filepath.Join(dataDir, "lifetime.txt") }
func modeFile() string     { return filepath.Join(dataDir, "mode.txt") }

type lifetimeData struct {
	Keys  int64 `json:"keys"`
	Left  int64 `json:"left"`
	Right int64 `json:"right"`
}

type stateJSON struct {
	Text     string `json:"text"`
	Mode     string `json:"mode"`
	Session  int64  `json:"session"`
	Lifetime int64  `json:"lifetime"`
}

func formatCount(n int64) string {
	if n < 0 {
		return "-" + formatCount(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
		b.WriteByte(',')
	}
	for i := remainder; i < len(s); i += 3 {
		if i > remainder {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func loadLifetime() lifetimeData {
	data, err := os.ReadFile(lifetimeFile())
	if err != nil {
		return lifetimeData{}
	}
	var ld lifetimeData
	if err := json.Unmarshal(data, &ld); err != nil {
		return lifetimeData{}
	}
	return ld
}

func saveLifetime(ld lifetimeData) {
	data, _ := json.Marshal(ld)
	tmp := lifetimeFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	os.Rename(tmp, lifetimeFile())
}

func loadMode() string {
	data, err := os.ReadFile(modeFile())
	if err != nil {
		return "session"
	}
	mode := strings.TrimSpace(string(data))
	if mode == "session" || mode == "lifetime" {
		return mode
	}
	return "session"
}

func saveMode(mode string) {
	os.WriteFile(modeFile(), []byte(mode), 0644)
}

func writeState(mode string, session int64, lt lifetimeData, leftClicks, rightClicks int64) {
	fc := formatCount
	keyWidth := max(len(fc(session)), len(fc(lt.Keys)))
	clickWidth := max(len(fc(leftClicks)), len(fc(lt.Left)))
	rclickWidth := max(len(fc(rightClicks)), len(fc(lt.Right)))

	var text string
	if mode == "session" {
		text = fmt.Sprintf("\u23f3 %*s \U0001f5b1 %*s|%*s",
			keyWidth, fc(session),
			clickWidth, fc(leftClicks),
			rclickWidth, fc(rightClicks))
	} else {
		text = fmt.Sprintf("\u267e\ufe0f %*s \U0001f5b1 %*s|%*s",
			keyWidth, fc(lt.Keys),
			clickWidth, fc(lt.Left),
			rclickWidth, fc(lt.Right))
	}

	state := stateJSON{
		Text:     text,
		Mode:     mode,
		Session:  session,
		Lifetime: lt.Keys,
	}
	data, _ := json.Marshal(state)
	tmp := stateFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	os.Rename(tmp, stateFile())
}

func signalWaybar() {
	exec.Command("pkill", fmt.Sprintf("-RTMIN+%d", waybarSignal), "waybar").Run()
}

func main() {
	log.SetFlags(0)
	os.MkdirAll(dataDir, 0755)

	keyboards, mice := evdev.DetectDevices()
	if len(keyboards) == 0 {
		log.Fatal("No keyboard devices detected")
	}
	if len(mice) == 0 {
		log.Fatal("No mouse devices detected")
	}
	for _, d := range keyboards {
		log.Printf("keyboard: %s (%s)", d.Name, d.Path)
	}
	for _, d := range mice {
		log.Printf("mouse:    %s (%s)", d.Name, d.Path)
	}

	lt := loadLifetime()
	mode := loadMode()
	var session, leftClicks, rightClicks int64
	lastSave := time.Now()

	writeState(mode, session, lt, leftClicks, rightClicks)
	signalWaybar()

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

	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, syscall.SIGTERM, syscall.SIGINT)

	sigToggle := make(chan os.Signal, 1)
	signal.Notify(sigToggle, syscall.SIGUSR1)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	cleanup := func() {
		saveLifetime(lt)
		writeState(mode, session, lt, leftClicks, rightClicks)
		signalWaybar()
	}

	for {
		select {
		case <-sigStop:
			cleanup()
			return

		case <-sigToggle:
			if mode == "session" {
				mode = "lifetime"
			} else {
				mode = "session"
			}
			saveMode(mode)
			writeState(mode, session, lt, leftClicks, rightClicks)
			signalWaybar()

		case ev := <-events:
			switch ev.Code {
			case evdev.BtnLeft:
				leftClicks++
				lt.Left++
			case evdev.BtnRight:
				rightClicks++
				lt.Right++
			default:
				session++
				lt.Keys++
			}
			writeState(mode, session, lt, leftClicks, rightClicks)
			signalWaybar()

		case <-ticker.C:
			if time.Since(lastSave) >= saveInterval {
				saveLifetime(lt)
				lastSave = time.Now()
			}
		}
	}
}
```

- [ ] **Step 3: Delete root `evdev.go` and root `main.go`**

```bash
rm evdev.go main.go
```

- [ ] **Step 4: Verify waybar binary builds**

Run: `go build -o bongocat-waybar ./cmd/waybar`
Expected: builds successfully with no errors

- [ ] **Step 5: Commit**

```bash
git add internal/evdev/evdev.go cmd/waybar/main.go
git rm evdev.go main.go
git commit -m "refactor: move to cmd/ layout with shared internal/evdev package"
```

---

### Task 2: Animation state machine (TDD)

Pure Go state machine with no GTK dependencies. Alternates left/right paw on keystrokes, maps mouse buttons directly.

**Files:**
- Create: `cmd/overlay/state_test.go`
- Create: `cmd/overlay/state.go`

- [ ] **Step 1: Write failing tests for state machine**

```go
package main

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := newState()
	if s.current() != stateIdle {
		t.Errorf("initial state = %v, want idle", s.current())
	}
}

func TestKeyAlternation(t *testing.T) {
	s := newState()

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Errorf("first key = %v, want left_paw", s.current())
	}

	s.onKey()
	if s.current() != stateRightPaw {
		t.Errorf("second key = %v, want right_paw", s.current())
	}

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Errorf("third key = %v, want left_paw", s.current())
	}
}

func TestMouseButtons(t *testing.T) {
	s := newState()

	s.onLeftClick()
	if s.current() != stateLeftPaw {
		t.Errorf("left click = %v, want left_paw", s.current())
	}

	s.onRightClick()
	if s.current() != stateRightPaw {
		t.Errorf("right click = %v, want right_paw", s.current())
	}
}

func TestBothPaws(t *testing.T) {
	s := newState()

	s.onLeftClick()
	s.onRightClick()
	if s.current() != stateBothPaws {
		t.Errorf("left+right = %v, want both_paws", s.current())
	}
}

func TestIdleTimeout(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond // short for testing

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Fatalf("after key = %v, want left_paw", s.current())
	}

	time.Sleep(100 * time.Millisecond)
	s.tick()
	if s.current() != stateIdle {
		t.Errorf("after timeout = %v, want idle", s.current())
	}
}

func TestNoIdleWhileActive(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond

	s.onKey()
	time.Sleep(30 * time.Millisecond)
	s.onKey() // resets timer
	time.Sleep(30 * time.Millisecond)
	s.tick()
	if s.current() == stateIdle {
		t.Error("should not be idle — input was recent")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cmd/overlay && go test -v -run TestNewState`
Expected: FAIL — `newState` not defined

- [ ] **Step 3: Implement state machine**

```go
package main

import "time"

type animState int

const (
	stateIdle animState = iota
	stateLeftPaw
	stateRightPaw
	stateBothPaws
)

func (s animState) String() string {
	switch s {
	case stateIdle:
		return "idle"
	case stateLeftPaw:
		return "left_paw"
	case stateRightPaw:
		return "right_paw"
	case stateBothPaws:
		return "both_paws"
	default:
		return "unknown"
	}
}

type state struct {
	st          animState
	nextPaw     animState // alternates between leftPaw and rightPaw for keys
	lastInput   time.Time
	idleTimeout time.Duration
	leftHeld    bool
	rightHeld   bool
}

func newState() *state {
	return &state{
		st:          stateIdle,
		nextPaw:     stateLeftPaw,
		idleTimeout: 200 * time.Millisecond,
	}
}

func (s *state) current() animState {
	return s.st
}

func (s *state) onKey() {
	s.lastInput = time.Now()
	s.st = s.nextPaw
	if s.nextPaw == stateLeftPaw {
		s.nextPaw = stateRightPaw
	} else {
		s.nextPaw = stateLeftPaw
	}
}

func (s *state) onLeftClick() {
	s.lastInput = time.Now()
	s.leftHeld = true
	if s.rightHeld {
		s.st = stateBothPaws
	} else {
		s.st = stateLeftPaw
	}
}

func (s *state) onRightClick() {
	s.lastInput = time.Now()
	s.rightHeld = true
	if s.leftHeld {
		s.st = stateBothPaws
	} else {
		s.st = stateRightPaw
	}
}

func (s *state) tick() {
	if s.st != stateIdle && time.Since(s.lastInput) >= s.idleTimeout {
		s.st = stateIdle
		s.leftHeld = false
		s.rightHeld = false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cmd/overlay && go test -v`
Expected: all 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/overlay/state.go cmd/overlay/state_test.go
git commit -m "feat: add animation state machine with paw alternation"
```

---

### Task 3: Manifest loading (TDD)

Parse the JSON manifest that describes sprite sheet layout.

**Files:**
- Create: `cmd/overlay/manifest_test.go`
- Create: `cmd/overlay/manifest.go`

- [ ] **Step 1: Write failing tests for manifest loading**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	data := `{
		"name": "testcat",
		"frame_width": 128,
		"frame_height": 128,
		"states": {
			"idle":      {"row": 0, "frames": 1, "fps": 1},
			"left_paw":  {"row": 1, "frames": 2, "fps": 8},
			"right_paw": {"row": 2, "frames": 2, "fps": 8},
			"both_paws": {"row": 3, "frames": 2, "fps": 8}
		}
	}`
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(data), 0644)

	m, err := loadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "testcat" {
		t.Errorf("name = %q, want testcat", m.Name)
	}
	if m.FrameWidth != 128 || m.FrameHeight != 128 {
		t.Errorf("frame size = %dx%d, want 128x128", m.FrameWidth, m.FrameHeight)
	}
	if len(m.States) != 4 {
		t.Errorf("states count = %d, want 4", len(m.States))
	}
	idle := m.States["idle"]
	if idle.Row != 0 || idle.Frames != 1 || idle.FPS != 1 {
		t.Errorf("idle state = %+v", idle)
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	_, err := loadManifest(t.TempDir())
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestManifestSpriteSheetPath(t *testing.T) {
	dir := t.TempDir()
	data := `{"name":"test","frame_width":64,"frame_height":64,"states":{}}`
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(data), 0644)

	m, err := loadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(dir, "sprite.png")
	if m.SpritePath != expected {
		t.Errorf("sprite path = %q, want %q", m.SpritePath, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cmd/overlay && go test -v -run TestLoadManifest`
Expected: FAIL — `loadManifest` not defined

- [ ] **Step 3: Implement manifest loading**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type stateSpec struct {
	Row    int `json:"row"`
	Frames int `json:"frames"`
	FPS    int `json:"fps"`
}

type manifest struct {
	Name        string               `json:"name"`
	FrameWidth  int                  `json:"frame_width"`
	FrameHeight int                  `json:"frame_height"`
	States      map[string]stateSpec `json:"states"`
	SpritePath  string               `json:"-"`
}

func loadManifest(dir string) (*manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	m.SpritePath = filepath.Join(dir, "sprite.png")
	return &m, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cmd/overlay && go test -v`
Expected: all tests PASS (state + manifest)

- [ ] **Step 5: Commit**

```bash
git add cmd/overlay/manifest.go cmd/overlay/manifest_test.go
git commit -m "feat: add sprite sheet manifest loading"
```

---

### Task 4: GTK4 window scaffolding

Set up the overlay entry point with a transparent GTK4 window. This task gets a window on screen — rendering comes next.

**Files:**
- Create: `cmd/overlay/main.go`
- Modify: `go.mod` (add gotk4 dependency)

- [ ] **Step 1: Add gotk4 dependency**

```bash
cd /home/chazzy/projects/bongocat
go get github.com/diamondburned/gotk4/pkg/gtk/v4
go get github.com/diamondburned/gotk4/pkg/gdk/v4
go get github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2
go get github.com/diamondburned/gotk4/pkg/core/glib
go get github.com/diamondburned/gotk4/pkg/cairo
```

- [ ] **Step 2: Create `cmd/overlay/main.go`**

Minimal GTK4 app with transparent window and a DrawingArea placeholder. No rendering yet — just proves the window works.

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var animDir = filepath.Join(os.Getenv("HOME"), ".config", "bongocat", "animations")

func main() {
	log.SetFlags(0)

	animName := flag.String("animation", "bongocat", "animation set name")
	sizeOverride := flag.Int("size", 0, "override window size in pixels")
	flag.Parse()

	mf, err := loadManifest(filepath.Join(animDir, *animName))
	if err != nil {
		log.Fatalf("load manifest: %v", err)
	}

	winSize := mf.FrameWidth
	if *sizeOverride > 0 {
		winSize = *sizeOverride
	}

	app := gtk.NewApplication("bongocat", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		win := gtk.NewApplicationWindow(app)
		win.SetTitle("bongocat")
		win.SetDefaultSize(winSize, winSize)
		win.SetDecorated(false)

		// Transparent background via CSS
		css := gtk.NewCSSProvider()
		css.LoadFromString("window, drawingarea { background-color: transparent; }")
		gtk.StyleContextAddProviderForDisplay(
			gdk.DisplayGetDefault(),
			css,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)

		area := gtk.NewDrawingArea()
		area.SetContentWidth(winSize)
		area.SetContentHeight(winSize)
		area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
			// placeholder — solid red square to prove it works
			cr.SetSourceRGBA(1, 0, 0, 0.5)
			cr.Rectangle(0, 0, float64(w), float64(h))
			cr.Fill()
		})

		win.SetChild(area)
		win.Show()

		fmt.Printf("overlay window: %dx%d (animation: %s)\n", winSize, winSize, mf.Name)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
```

Note: This will show a semi-transparent red square. We replace the draw func with real sprite rendering in Task 5.

- [ ] **Step 3: Verify it builds**

Run: `go build -o bongocat-overlay ./cmd/overlay`
Expected: builds successfully (needs GTK4 dev headers installed: `sudo pacman -S gtk4`)

- [ ] **Step 4: Create a test manifest and test manually**

```bash
mkdir -p ~/.config/bongocat/animations/bongocat
cat > ~/.config/bongocat/animations/bongocat/manifest.json << 'EOF'
{
  "name": "bongocat",
  "frame_width": 128,
  "frame_height": 128,
  "states": {
    "idle":      {"row": 0, "frames": 1, "fps": 1},
    "left_paw":  {"row": 1, "frames": 2, "fps": 8},
    "right_paw": {"row": 2, "frames": 2, "fps": 8},
    "both_paws": {"row": 3, "frames": 2, "fps": 8}
  }
}
EOF
```

Run: `./bongocat-overlay`
Expected: a small 128x128 semi-transparent red square window appears, draggable

- [ ] **Step 5: Commit**

```bash
git add cmd/overlay/main.go go.mod go.sum
git commit -m "feat: scaffold GTK4 overlay window with transparency"
```

---

### Task 5: Sprite sheet rendering

Load the sprite sheet, crop frames by state, and draw them in the DrawingArea.

**Files:**
- Create: `cmd/overlay/render.go`
- Modify: `cmd/overlay/main.go` (wire renderer into draw func)

- [ ] **Step 1: Implement renderer**

```go
package main

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
)

type renderer struct {
	sheet    *gdkpixbuf.Pixbuf
	manifest *manifest
	frame    int // current frame index within current state animation
}

func newRenderer(mf *manifest) (*renderer, error) {
	sheet, err := gdkpixbuf.NewPixbufFromFile(mf.SpritePath)
	if err != nil {
		return nil, fmt.Errorf("load sprite sheet %s: %w", mf.SpritePath, err)
	}
	return &renderer{sheet: sheet, manifest: mf}, nil
}

// draw renders the current animation frame for the given state into the cairo context.
func (r *renderer) draw(cr *cairo.Context, st animState, w, h int) {
	spec, ok := r.manifest.States[st.String()]
	if !ok {
		return
	}

	frameIdx := r.frame % spec.Frames
	srcX := frameIdx * r.manifest.FrameWidth
	srcY := spec.Row * r.manifest.FrameHeight

	sub := r.sheet.NewSubpixbuf(srcX, srcY, r.manifest.FrameWidth, r.manifest.FrameHeight)
	if sub == nil {
		return
	}

	// Scale to fill the drawing area if sizes differ
	if r.manifest.FrameWidth != w || r.manifest.FrameHeight != h {
		sub = sub.ScaleSimple(w, h, gdkpixbuf.InterpNearest)
	}

	gdk.CairoSetSourcePixbuf(cr, sub, 0, 0)
	cr.Paint()
}

// advance moves to the next frame in the current animation.
// Returns true if the frame actually changed (for redraw optimization).
func (r *renderer) advance() bool {
	old := r.frame
	r.frame++
	return r.frame != old
}

// reset resets the frame counter (called on state change).
func (r *renderer) reset() {
	r.frame = 0
}
```

- [ ] **Step 2: Update `cmd/overlay/main.go` draw func to use renderer**

Replace the placeholder red square draw func. Add the cairo import and wire in the renderer:

In `main.go`, replace the `area.SetDrawFunc` block and add a renderer variable:

```go
// After loading manifest, before app.ConnectActivate:
// (these become module-level or closure-captured variables)

// Inside ConnectActivate, after creating area:
r, err := newRenderer(mf)
if err != nil {
    log.Fatalf("init renderer: %v", err)
}

st := newState()

area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
    r.draw(cr, st.current(), w, h)
})
```

The full updated `main.go` ConnectActivate closure becomes:

```go
app.ConnectActivate(func() {
    win := gtk.NewApplicationWindow(app)
    win.SetTitle("bongocat")
    win.SetDefaultSize(winSize, winSize)
    win.SetDecorated(false)

    css := gtk.NewCSSProvider()
    css.LoadFromString("window, drawingarea { background-color: transparent; }")
    gtk.StyleContextAddProviderForDisplay(
        gdk.DisplayGetDefault(),
        css,
        uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
    )

    rend, err := newRenderer(mf)
    if err != nil {
        log.Fatalf("init renderer: %v", err)
    }

    st := newState()

    area := gtk.NewDrawingArea()
    area.SetContentWidth(winSize)
    area.SetContentHeight(winSize)
    area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
        rend.draw(cr, st.current(), w, h)
    })

    win.SetChild(area)
    win.Show()

    fmt.Printf("overlay window: %dx%d (animation: %s)\n", winSize, winSize, mf.Name)
})
```

- [ ] **Step 3: Verify it builds**

Run: `go build -o bongocat-overlay ./cmd/overlay`
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add cmd/overlay/render.go cmd/overlay/main.go
git commit -m "feat: add sprite sheet renderer with frame cropping"
```

---

### Task 6: Wire evdev input to state machine and renderer

Connect keyboard/mouse events from evdev to the state machine, drive animation frame advancement with a GLib timer, and redraw on state changes.

**Files:**
- Modify: `cmd/overlay/main.go`

- [ ] **Step 1: Add evdev event loop and GLib timer to main.go**

Replace the full `main.go` with the wired-up version:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/chazzy/bongocat/internal/evdev"
)

var animDir = filepath.Join(os.Getenv("HOME"), ".config", "bongocat", "animations")

func main() {
	log.SetFlags(0)

	animName := flag.String("animation", "bongocat", "animation set name")
	sizeOverride := flag.Int("size", 0, "override window size in pixels")
	flag.Parse()

	mf, err := loadManifest(filepath.Join(animDir, *animName))
	if err != nil {
		log.Fatalf("load manifest: %v", err)
	}

	winSize := mf.FrameWidth
	if *sizeOverride > 0 {
		winSize = *sizeOverride
	}

	// Detect input devices
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

	app := gtk.NewApplication("bongocat", gio.ApplicationFlagsNone)

	app.ConnectActivate(func() {
		win := gtk.NewApplicationWindow(app)
		win.SetTitle("bongocat")
		win.SetDefaultSize(winSize, winSize)
		win.SetDecorated(false)

		css := gtk.NewCSSProvider()
		css.LoadFromString("window, drawingarea { background-color: transparent; }")
		gtk.StyleContextAddProviderForDisplay(
			gdk.DisplayGetDefault(),
			css,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)

		rend, err := newRenderer(mf)
		if err != nil {
			log.Fatalf("init renderer: %v", err)
		}

		st := newState()

		area := gtk.NewDrawingArea()
		area.SetContentWidth(winSize)
		area.SetContentHeight(winSize)
		area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
			rend.draw(cr, st.current(), w, h)
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
					prev := st.current()
					switch ev.Code {
					case evdev.BtnLeft:
						st.onLeftClick()
					case evdev.BtnRight:
						st.onRightClick()
					default:
						st.onKey()
					}
					if st.current() != prev {
						rend.reset()
					}
					area.QueueDraw()
				})
			}
		}()

		// Animation frame timer + idle check (~60 ticks/sec)
		glib.TimeoutAdd(16, func() bool {
			prev := st.current()
			st.tick()
			if st.current() != prev {
				rend.reset()
				area.QueueDraw()
			} else if st.current() != stateIdle {
				rend.advance()
				area.QueueDraw()
			}
			return true // keep timer running
		})

		win.Show()
		fmt.Printf("overlay: %dx%d animation=%s\n", winSize, winSize, mf.Name)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build -o bongocat-overlay ./cmd/overlay`
Expected: builds successfully

- [ ] **Step 3: Test manually (requires a sprite.png)**

Create a placeholder 256x512 PNG with colored rectangles for each state row, place it at `~/.config/bongocat/animations/bongocat/sprite.png`, and run:

```bash
./bongocat-overlay
```

Expected: window appears, typing changes the displayed sprite row, mouse clicks change it differently, idle timeout returns to row 0.

- [ ] **Step 4: Commit**

```bash
git add cmd/overlay/main.go
git commit -m "feat: wire evdev input to state machine and GTK renderer"
```

---

### Task 7: Config files and default animation

Create the systemd service, example Hyprland rules, and default manifest.

**Files:**
- Create: `config/bongocat-overlay.service`
- Create: `config/hyprland.conf`
- Create: `animations/bongocat/manifest.json`

- [ ] **Step 1: Create systemd service**

```ini
[Unit]
Description=Bongocat Overlay
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.config/bongocat/bongocat-overlay
Restart=on-failure
RestartSec=3

[Install]
WantedBy=graphical-session.target
```

- [ ] **Step 2: Create example Hyprland rules**

```
# Bongocat overlay window rules
# Add these to your ~/.config/hypr/hyprland.conf

windowrulev2 = float, class:^(bongocat)$
windowrulev2 = pin, class:^(bongocat)$
windowrulev2 = noborder, class:^(bongocat)$
windowrulev2 = noshadow, class:^(bongocat)$
windowrulev2 = noblur, class:^(bongocat)$
windowrulev2 = size 128 128, class:^(bongocat)$

# Toggle visibility
bind = SUPER SHIFT, B, exec, pkill -USR2 bongocat-overlay
```

- [ ] **Step 3: Create default manifest**

```json
{
  "name": "bongocat",
  "frame_width": 128,
  "frame_height": 128,
  "states": {
    "idle":      { "row": 0, "frames": 1, "fps": 1 },
    "left_paw":  { "row": 1, "frames": 2, "fps": 8 },
    "right_paw": { "row": 2, "frames": 2, "fps": 8 },
    "both_paws": { "row": 3, "frames": 2, "fps": 8 }
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add config/bongocat-overlay.service config/hyprland.conf animations/bongocat/manifest.json
git commit -m "feat: add overlay systemd service, hyprland rules, and default manifest"
```

---

### Task 8: Update install script

Update `install.sh` to build and install both binaries, copy animations, and install the overlay service.

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Update install.sh**

```bash
#!/bin/bash
# Build and install bongocat (Go version)
#
# Usage: ./install.sh            — build, install, enable services
#        ./install.sh --uninstall — stop, disable, remove

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_DIR="$HOME/.config/bongocat"
WAYBAR_SERVICE="$HOME/.config/systemd/user/bongocat-waybar.service"
OVERLAY_SERVICE="$HOME/.config/systemd/user/bongocat-overlay.service"

if [[ "${1:-}" == "--uninstall" ]]; then
    systemctl --user disable --now bongocat-waybar.service 2>/dev/null || true
    systemctl --user disable --now bongocat-overlay.service 2>/dev/null || true
    rm -f "$WAYBAR_SERVICE" "$OVERLAY_SERVICE"
    systemctl --user daemon-reload
    echo "Uninstalled. Remove waybar/hyprland config manually if desired."
    exit 0
fi

# Build
echo "Building bongocat-waybar..."
cd "$PROJECT_DIR"
go build -o bongocat-waybar ./cmd/waybar
echo "Built: $PROJECT_DIR/bongocat-waybar"

echo "Building bongocat-overlay..."
go build -o bongocat-overlay ./cmd/overlay
echo "Built: $PROJECT_DIR/bongocat-overlay"

# Stop running services before overwriting binaries
systemctl --user stop bongocat-waybar.service 2>/dev/null || true
systemctl --user stop bongocat-overlay.service 2>/dev/null || true

# Copy to config dir
mkdir -p "$CONFIG_DIR"
cp bongocat-waybar "$CONFIG_DIR/bongocat-waybar"
cp bongocat-overlay "$CONFIG_DIR/bongocat-overlay"
cp config/bongocat-waybar.service "$CONFIG_DIR/bongocat-waybar.service"
cp config/bongocat-overlay.service "$CONFIG_DIR/bongocat-overlay.service"
cp config/waybar.jsonc "$CONFIG_DIR/waybar.jsonc"
cp config/waybar.css "$CONFIG_DIR/waybar.css"
cp config/hyprland.conf "$CONFIG_DIR/hyprland.conf"

# Copy default animation
mkdir -p "$CONFIG_DIR/animations/bongocat"
cp animations/bongocat/manifest.json "$CONFIG_DIR/animations/bongocat/manifest.json"
if [ -f animations/bongocat/sprite.png ]; then
    cp animations/bongocat/sprite.png "$CONFIG_DIR/animations/bongocat/sprite.png"
fi
echo "Installed files to $CONFIG_DIR"

# Symlink and enable systemd services
mkdir -p "$(dirname "$WAYBAR_SERVICE")"
ln -sf "$CONFIG_DIR/bongocat-waybar.service" "$WAYBAR_SERVICE"
ln -sf "$CONFIG_DIR/bongocat-overlay.service" "$OVERLAY_SERVICE"
systemctl --user daemon-reload
systemctl --user enable --now bongocat-waybar.service
systemctl --user enable --now bongocat-overlay.service
echo "Services enabled and started."

echo ""
echo "Add 'group/bongocat' to modules-center in ~/.config/waybar/config.jsonc"
echo "and copy module definitions from $CONFIG_DIR/waybar.jsonc"
echo "and styles from $CONFIG_DIR/waybar.css into ~/.config/waybar/style.css"
echo ""
echo "Add window rules from $CONFIG_DIR/hyprland.conf to ~/.config/hypr/hyprland.conf"
echo ""
echo "NOTE: You need a sprite.png at $CONFIG_DIR/animations/bongocat/sprite.png"
echo "  256x512 PNG, 4 rows x 2 columns of 128x128 frames"
echo "  Row 0: idle, Row 1: left paw, Row 2: right paw, Row 3: both paws"
```

- [ ] **Step 2: Verify the script is syntactically valid**

Run: `bash -n install.sh`
Expected: no output (no syntax errors)

- [ ] **Step 3: Commit**

```bash
git add install.sh
git commit -m "feat: update install script for overlay binary and animations"
```

---

### Task 9: Create default sprite sheet

The overlay needs a `sprite.png` to render. This is a 256x512 pixel art image.

**Files:**
- Create: `animations/bongocat/sprite.png`

- [ ] **Step 1: Create or source a bongocat sprite sheet**

This is a manual/creative step. The sprite sheet layout is:
- 256px wide (2 columns of 128x128 frames)
- 512px tall (4 rows)
- Row 0: idle cat (1 frame, col 0)
- Row 1: left paw down (2 frames, cols 0-1)
- Row 2: right paw down (2 frames, cols 0-1)
- Row 3: both paws down (2 frames, cols 0-1)
- Transparent background (PNG with alpha)

Options:
- Draw it in a pixel art editor (Aseprite, Piskel, GIMP)
- Adapt an existing bongocat sprite (many open-source versions exist)
- Use a placeholder with colored rectangles for initial testing

Place the final file at `animations/bongocat/sprite.png`.

- [ ] **Step 2: Commit**

```bash
git add animations/bongocat/sprite.png
git commit -m "feat: add default bongocat sprite sheet"
```
