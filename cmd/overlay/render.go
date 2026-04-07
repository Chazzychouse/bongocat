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
	y := float64(h) * 0.45

	for i, entry := range entries {
		if entry.isMouse {
			r.drawMouse(cr, entry.name, float64(w), y, fontSize, opacity)
		} else {
			r.drawKey(cr, entry.name, float64(w), y, fontSize, opacity)
		}

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
	// Shrink font if text is wider than 90% of the window
	maxW := w * 0.9
	if ext.Width > maxW {
		fontSize *= maxW / ext.Width
		cr.SetFontSize(fontSize)
		ext = cr.TextExtents(name)
	}
	x := (w - ext.Width) / 2 - ext.XBearing
	cr.MoveTo(x, y)
	cr.SetSourceRGBA(1, 1, 1, opacity)
	cr.ShowText(name)
}

func (r *renderer) drawMouse(cr *cairo.Context, name string, w, y, fontSize, opacity float64) {
	mw := fontSize * 0.5
	mh := fontSize * 0.75
	x := (w - mw) / 2
	top := y - mh*0.7
	radius := mw * 0.25

	cr.SetLineWidth(2)

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

	mid := x + mw/2
	split := top + mh*0.45
	cr.MoveTo(mid, top+radius*0.5)
	cr.LineTo(mid, split)
	cr.SetSourceRGBA(1, 1, 1, opacity)
	cr.Stroke()

	cr.MoveTo(x+radius*0.5, split)
	cr.LineTo(x+mw-radius*0.5, split)
	cr.Stroke()

	if name == "LMB" {
		cr.Rectangle(x+radius*0.5, top+radius*0.5, mw/2-radius*0.5, split-top-radius*0.5)
		cr.SetSourceRGBA(1, 1, 1, opacity*0.8)
		cr.Fill()
	} else if name == "RMB" {
		cr.Rectangle(mid, top+radius*0.5, mw/2-radius*0.5, split-top-radius*0.5)
		cr.SetSourceRGBA(1, 1, 1, opacity*0.8)
		cr.Fill()
	}
}
