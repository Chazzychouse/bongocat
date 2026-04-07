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
