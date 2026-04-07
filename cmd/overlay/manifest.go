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
