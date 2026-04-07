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
	frame    int
}

func newRenderer(mf *manifest) (*renderer, error) {
	sheet, err := gdkpixbuf.NewPixbufFromFile(mf.SpritePath)
	if err != nil {
		return nil, fmt.Errorf("load sprite sheet %s: %w", mf.SpritePath, err)
	}
	return &renderer{sheet: sheet, manifest: mf}, nil
}

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

	if r.manifest.FrameWidth != w || r.manifest.FrameHeight != h {
		sub = sub.ScaleSimple(w, h, gdkpixbuf.InterpNearest)
	}

	gdk.CairoSetSourcePixbuf(cr, sub, 0, 0)
	cr.Paint()
}

func (r *renderer) advance() bool {
	old := r.frame
	r.frame++
	return r.frame != old
}

func (r *renderer) reset() {
	r.frame = 0
}
