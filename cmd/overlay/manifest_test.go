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
