# Bongocat Overlay — Design Spec

## Overview

A GTK4 overlay window that displays an animated bongo cat reacting to keyboard and mouse input. Runs as a separate binary (`bongocat-overlay`) alongside the existing `bongocat-waybar` counter. The window is a normal GTK4 window styled via Hyprland window rules to float, pin, and strip decorations — no layer-shell dependency.

## Architecture

### Two-binary layout

Both binaries live under `cmd/` following standard Go multi-binary convention:

```
bongocat/
  evdev.go                       # shared input detection (existing)
  cmd/
    waybar/
      main.go                    # existing waybar counter (moved from root)
    overlay/
      main.go                    # GTK4 window setup, signal handling, event loop
      render.go                  # sprite sheet loading, frame cropping, drawing
      state.go                   # state machine (idle/left/right/both, alternation)
  animations/
    bongocat/
      sprite.png                 # default bongocat sprite sheet
      manifest.json              # frame layout and timing
  config/
    bongocat-waybar.service      # existing systemd unit
    bongocat-overlay.service     # new systemd unit
    hyprland.conf                # example Hyprland window rules
    waybar.jsonc                 # existing
    waybar.css                   # existing
  install.sh                     # updated for both binaries
  go.mod
  go.sum
```

Build commands:
- `go build -o bongocat-waybar ./cmd/waybar`
- `go build -o bongocat-overlay ./cmd/overlay`

### Independence

The two binaries are fully independent. Each opens evdev devices directly (the kernel allows multiple readers). No IPC between them. You can run either or both.

## Animation System

### Directory structure

Animations live in `~/.config/bongocat/animations/`. Each animation set is a subdirectory containing a sprite sheet and manifest:

```
~/.config/bongocat/animations/
  bongocat/
    sprite.png
    manifest.json
  nyancat/
    sprite.png
    manifest.json
```

### Manifest format

```json
{
  "name": "bongocat",
  "frame_width": 128,
  "frame_height": 128,
  "states": {
    "idle":       { "row": 0, "frames": 1, "fps": 1 },
    "left_paw":   { "row": 1, "frames": 2, "fps": 8 },
    "right_paw":  { "row": 2, "frames": 2, "fps": 8 },
    "both_paws":  { "row": 3, "frames": 2, "fps": 8 }
  }
}
```

- Each row in the sprite sheet corresponds to a state
- `frames` = number of columns (animation frames) for that state
- `fps` = playback speed for multi-frame states
- Frame dimensions define the window size by default

### Swapping animations

Drop a new directory into the animations folder and launch with `--animation <name>`. No code changes needed.

## Input & State Machine

### States

`idle`, `left_paw`, `right_paw`, `both_paws`

### Transitions

| Input | State |
|---|---|
| Key down | Alternates `left_paw` / `right_paw` each press |
| Left mouse click | `left_paw` |
| Right mouse click | `right_paw` |
| Left + right click in same frame | `both_paws` |
| No input for ~200ms | `idle` |

Keyboard alternation makes rapid typing look like the cat is bongo-ing with both paws.

### Input source

Reuses `evdev.go` from the shared package. The overlay opens `/dev/input/event*` devices independently, same as the waybar binary.

## GTK4 Window

### Setup

- **gotk4** Go bindings for GTK4 (CGo bridge)
- Window class: `bongocat` (for Hyprland rule targeting)
- RGBA visual with transparent background — only the sprite is visible
- GDK Pixbuf loads the sprite sheet at startup
- GLib timeout drives the render loop at the manifest's `fps`
- Input events arrive via channel from evdev goroutines; state updates dispatched to GTK main thread via `glib.IdleAdd()`

### Hyprland window rules

```
windowrulev2 = float, class:^(bongocat)$
windowrulev2 = pin, class:^(bongocat)$
windowrulev2 = noborder, class:^(bongocat)$
windowrulev2 = noshadow, class:^(bongocat)$
windowrulev2 = noblur, class:^(bongocat)$
windowrulev2 = size 128 128, class:^(bongocat)$
```

The window is freely draggable (normal GTK4 window behavior under Hyprland float rules).

### Visibility toggle

SIGUSR2 flips a visibility bool and calls `window.SetVisible()`. Bind in Hyprland config:

```
bind = SUPER SHIFT, B, exec, pkill -USR2 bongocat-overlay
```

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--animation` | `bongocat` | Animation set name (subdirectory under animations/) |
| `--size` | from manifest | Override window size in pixels |

## Systemd Service

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

## Install Script Updates

The existing `install.sh` is updated to:
- Build both binaries
- Copy `bongocat-overlay` to `~/.config/bongocat/`
- Copy default animation set to `~/.config/bongocat/animations/bongocat/`
- Install `bongocat-overlay.service` systemd unit
- Copy example Hyprland rules to `~/.config/bongocat/hyprland.conf`

## Default Sprite Sheet

A bongocat sprite sheet needs to be created or sourced with 4 rows:
- Row 0: idle (1 frame)
- Row 1: left paw down (2 frames)
- Row 2: right paw down (2 frames)
- Row 3: both paws down (2 frames)

Each frame is 128x128 pixels. The sprite sheet is 256x512 (2 columns x 4 rows).
