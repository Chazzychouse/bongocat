# bongocat-waybar

A lightweight Linux daemon that counts your keypresses and mouse clicks, displaying live stats in [Waybar](https://github.com/Alexays/Waybar).

Tracks per-session and lifetime (persisted across restarts) counts for keypresses, left clicks, and right clicks. Click the module to toggle between session and lifetime display.

## Requirements

- Linux with evdev (`/dev/input/event*`)
- Go 1.26+
- Waybar
- `jq`
- Read access to input devices (typically requires being in the `input` group)

## Install

```sh
git clone https://github.com/chazzychouse/bongocat.git
cd bongocat
./install.sh
```

This builds the binary, copies it and config files to `~/.config/bongocat/`, and enables a systemd user service.

Then add the module to your Waybar config:

1. Add `"group/bongocat"` to `modules-center` (or whichever section) in `~/.config/waybar/config.jsonc`
2. Copy the module definitions from `~/.config/bongocat/waybar.jsonc` into your Waybar config
3. Copy the styles from `~/.config/bongocat/waybar.css` into `~/.config/waybar/style.css`

## Uninstall

```sh
./install.sh --uninstall
```

## Usage

The daemon runs as a systemd user service and writes state to `~/.local/share/bongocat/state.json`.

- **Toggle session/lifetime display**: Click the Waybar module, or send `SIGUSR1`:
  ```sh
  pkill -USR1 bongocat-waybar
  ```

## How it works

- Reads raw input events from evdev devices via blocking reads
- Auto-detects keyboards (devices with alphabetic keys) and mice (devices with relative X/Y axes and left click)
- If a [Makima](https://github.com/cy8berpunk/makima) virtual keyboard is detected, it is used exclusively to avoid double-counting
- Lifetime counts are persisted to disk every 10 seconds and on shutdown
- Waybar is signaled via `SIGRTMIN+10` on every event for real-time updates
