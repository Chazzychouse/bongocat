#!/bin/bash
# Build and install bongocat-waybar (Go version)
#
# Usage: ./install.sh            — build, install, enable service
#        ./install.sh --uninstall — stop, disable, remove

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_DIR="$HOME/.config/bongocat"
SERVICE_DST="$HOME/.config/systemd/user/bongocat-waybar.service"

if [[ "${1:-}" == "--uninstall" ]]; then
    systemctl --user disable --now bongocat-waybar.service 2>/dev/null || true
    rm -f "$SERVICE_DST"
    systemctl --user daemon-reload
    echo "Uninstalled. Remove waybar module config manually if desired."
    exit 0
fi

# Build
echo "Building bongocat-waybar..."
cd "$PROJECT_DIR"
go build -o bongocat-waybar .
echo "Built: $PROJECT_DIR/bongocat-waybar"

# Stop running service before overwriting the binary
systemctl --user stop bongocat-waybar.service 2>/dev/null || true

# Copy to config dir
mkdir -p "$CONFIG_DIR"
cp bongocat-waybar "$CONFIG_DIR/bongocat-waybar"
cp config/bongocat-waybar.service "$CONFIG_DIR/bongocat-waybar.service"
cp config/waybar.jsonc "$CONFIG_DIR/waybar.jsonc"
cp config/waybar.css "$CONFIG_DIR/waybar.css"
echo "Installed files to $CONFIG_DIR"

# Symlink and enable systemd service
mkdir -p "$(dirname "$SERVICE_DST")"
ln -sf "$CONFIG_DIR/bongocat-waybar.service" "$SERVICE_DST"
systemctl --user daemon-reload
systemctl --user enable --now bongocat-waybar.service
echo "Service enabled and started."

echo ""
echo "Add 'group/bongocat' to modules-center in ~/.config/waybar/config.jsonc"
echo "and copy module definitions from $CONFIG_DIR/waybar.jsonc"
echo "and styles from $CONFIG_DIR/waybar.css into ~/.config/waybar/style.css"
