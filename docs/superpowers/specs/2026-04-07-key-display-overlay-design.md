# Key Display Overlay — Design Spec

## Overview

Replace the sprite-based bongocat overlay with a key/mouse display overlay that shows the actual keys and mouse buttons being pressed. The overlay renders a centered prominent key label with a trailing history of 3-4 previous keys that shrink and fade.

This is the "simple first pass" — a future iteration will render a full keyboard layout with highlighted keys.

## Approach

Cairo text rendering within the existing GTK4 DrawingArea draw callback. No new dependencies. Same evdev → state → render pipeline, with the state machine and renderer replaced.

## Components

### 1. Key Name Mapping (`internal/evdev/evdev.go`)

Add a `KeyName(code uint16) string` function that maps evdev key codes to human-readable display labels.

Coverage:
- Alphabetic keys: `KEY_A` (30) → "A" through `KEY_Z`
- Number keys: `KEY_1` (2) → "1" through `KEY_0` (11) → "0"
- Modifier keys: "Shift", "Ctrl", "Alt", "Super"
- Function keys: "F1" through "F12"
- Navigation: "Esc", "Tab", "Space", "Enter", "Backspace", "Delete", "Insert", "Home", "End", "PgUp", "PgDn"
- Arrow keys: "↑", "↓", "←", "→"
- Punctuation/symbols: mapped to their printed character
- Mouse buttons: `BTN_LEFT` (0x110) → "LMB", `BTN_RIGHT` (0x111) → "RMB"
- Unknown codes: formatted as hex, e.g. "0x1F"

This is a pure function with no side effects — a `map[uint16]string` lookup with a hex fallback.

### 2. State Machine (`cmd/overlay/state.go`)

Replace the paw-based state machine with a ring buffer of recent key entries.

```go
type keyEntry struct {
    name    string
    isMouse bool
    time    time.Time
}
```

**State fields:**
- `history []keyEntry` — ring buffer, capacity 5 (1 prominent + up to 4 trailing)
- `lastInput time.Time` — for idle timeout
- `idleTimeout time.Duration` — 200ms (unchanged)

**Methods:**
- `push(name string, isMouse bool)` — prepend to history, cap at 5, update lastInput
- `tick()` — if idle timeout exceeded, clear history
- `entries() []keyEntry` — return current history (newest first)
- `active() bool` — true if history is non-empty

The left/right paw states, nextPaw alternation, and held-button tracking are all removed.

### 3. Renderer (`cmd/overlay/render.go`)

Replace sprite rendering with Cairo text and path drawing.

**Key text rendering:**
- Newest key: centered in window, font size = ~50% of window height, full white (`rgba(1,1,1,1)`)
- Each trailing key drawn below the previous, with:
  - Font size: 70% of previous key's size
  - Opacity: 60% of previous key's opacity
  - Slight vertical offset downward
- Font: system monospace (Cairo toy font API with `"monospace"` family, `FONT_SLANT_NORMAL`, `FONT_WEIGHT_BOLD`)
- Text is centered using `TextExtents` to measure width/height

**Mouse icon rendering:**
- When an entry has `isMouse == true`, draw a mouse icon instead of text
- Mouse shape: rounded rectangle body (~60% width, 80% height of the text bounding box equivalent), two button regions on top half separated by a vertical line
- Clicked button (left or right, derived from name "LMB"/"RMB") filled white; other button outlined
- Scales and fades identically to text entries in the trail

**Renderer struct simplifies to:**
```go
type renderer struct {
    // No fields needed for simple version.
    // Future: could cache font metrics or mouse path.
}
```

**Draw function signature stays the same:** `draw(cr *cairo.Context, entries []keyEntry, w, h int)`

### 4. Main Loop (`cmd/overlay/main.go`)

**Remove:**
- Manifest loading (`loadManifest`, `-animation` flag)
- `animDir` variable
- Renderer initialization from manifest (`newRenderer(mf)`)
- Frame advance logic in the tick timer

**Change:**
- Event handler calls `st.push(evdev.KeyName(ev.Code), ev.Code == evdev.BtnLeft || ev.Code == evdev.BtnRight)` instead of `onKey/onLeftClick/onRightClick`
- Draw function calls `rend.draw(cr, st.entries(), w, h)`
- Default window size: 200×200 (no manifest to derive from)
- Keep `-size` flag for override
- Tick timer simplifies: just call `st.tick()`, if history changed → `area.QueueDraw()`

**Keep unchanged:**
- evdev device detection and goroutine-per-device reading
- GTK4 app setup, transparent window, CSS
- SIGUSR2 visibility toggle
- `glib.IdleAdd` dispatch pattern
- ~60fps tick timer

### 5. Delete `cmd/overlay/manifest.go`

No longer needed. The manifest and sprite sheet system is fully replaced.

### 6. Tests

**Update `cmd/overlay/state_test.go`:**
- Test `push` adds entries in order
- Test ring buffer caps at 5
- Test `tick` clears history after idle timeout
- Test `entries` returns newest-first order

**Delete `cmd/overlay/manifest_test.go`:**
- No longer applicable

**Add `internal/evdev/keyname_test.go`:**
- Test known key codes map correctly
- Test unknown codes produce hex string
- Test mouse button codes

## Data Flow

```
/dev/input/event* → evdev.ReadDevice goroutines → KeyEvent channel
    → glib.IdleAdd: evdev.KeyName(code) → state.push(name, isMouse)
    → area.QueueDraw()
    → draw callback: state.entries() → Cairo text/icon render
```

## Window Behavior

- Transparent background (unchanged)
- No decorations (unchanged)
- Default 200×200, overridable with `-size`
- SIGUSR2 toggles visibility (unchanged)
- Idle: window stays visible but empty (nothing drawn when history is empty)

## Future Work (not in scope)

- Full keyboard layout rendering with spatial key positions
- Key-up tracking for "currently held" display
- Configurable fonts, colors, trail length
- Combo detection (e.g., "Ctrl+C" displayed as one unit)
