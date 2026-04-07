// Command gen-sprite generates the default bongocat sprite sheet.
//
// Output: animations/bongocat/sprite.png (256x512, 4 rows x 2 cols of 128x128 frames)
//
// Row 0: idle (1 frame)
// Row 1: left paw down (2 frames)
// Row 2: right paw down (2 frames)
// Row 3: both paws down (2 frames)
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const (
	fw = 128 // frame width
	fh = 128 // frame height
	sw = fw * 2
	sh = fh * 4
)

var (
	white    = color.NRGBA{255, 255, 255, 255}
	black    = color.NRGBA{0, 0, 0, 255}
	pink     = color.NRGBA{255, 180, 180, 255}
	blush    = color.NRGBA{255, 150, 150, 120}
	eyeColor = color.NRGBA{40, 40, 40, 255}
	table    = color.NRGBA{160, 130, 100, 255}
	tableLt  = color.NRGBA{180, 150, 120, 255}
	padPink  = color.NRGBA{255, 160, 160, 255}
)

func main() {
	log.SetFlags(0)
	img := image.NewNRGBA(image.Rect(0, 0, sw, sh))

	// Row 0: idle
	drawFrame(img, 0, 0, false, false)
	drawFrame(img, 1, 0, false, false) // duplicate for modulo

	// Row 1: left paw animation (frame0=raised, frame1=down)
	drawFrame(img, 0, 1, false, false) // left raised (same as idle visually)
	drawFrame(img, 1, 1, true, false)  // left down

	// Row 2: right paw animation
	drawFrame(img, 0, 2, false, false) // right raised
	drawFrame(img, 1, 2, false, true)  // right down

	// Row 3: both paws animation
	drawFrame(img, 0, 3, false, false) // both raised
	drawFrame(img, 1, 3, true, true)   // both down

	out := filepath.Join("animations", "bongocat", "sprite.png")
	os.MkdirAll(filepath.Dir(out), 0755)
	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		log.Fatal(err)
	}
	f.Close()
	log.Printf("wrote %s (%dx%d)", out, sw, sh)
}

func drawFrame(img *image.NRGBA, col, row int, leftDown, rightDown bool) {
	ox := col * fw
	oy := row * fh

	// Table surface — flat across bottom third
	fillRect(img, ox, oy+92, ox+fw, oy+fh, table)
	fillRect(img, ox, oy+92, ox+fw, oy+95, tableLt) // highlight edge

	// Body — oval behind table, cat leaning forward
	fillEllipse(img, ox+64, oy+68, 32, 28, white)
	drawOutline(img, ox+64, oy+68, 32, 28, black, 1.8)

	// Head — large round head, the star of the show
	fillEllipse(img, ox+64, oy+38, 28, 25, white)

	// Ears (triangles poking up from head)
	drawFilledEar(img, ox+40, oy+16, true)
	drawFilledEar(img, ox+88, oy+16, false)

	// Head outline on top of ears
	drawOutline(img, ox+64, oy+38, 28, 25, black, 1.8)

	// Eyes — big, cute, slightly apart
	fillEllipse(img, ox+52, oy+36, 4, 5, eyeColor)
	fillEllipse(img, ox+76, oy+36, 4, 5, eyeColor)
	// Highlights
	fillEllipse(img, ox+53, oy+34, 2, 2, white)
	fillEllipse(img, ox+77, oy+34, 2, 2, white)

	// Nose — tiny pink triangle
	fillEllipse(img, ox+64, oy+43, 2, 2, pink)

	// Mouth — small w shape
	setPixel(img, ox+60, oy+46, black)
	setPixel(img, ox+61, oy+47, black)
	setPixel(img, ox+62, oy+48, black)
	setPixel(img, ox+63, oy+47, black)
	setPixel(img, ox+64, oy+48, black)
	setPixel(img, ox+65, oy+47, black)
	setPixel(img, ox+66, oy+48, black)
	setPixel(img, ox+67, oy+47, black)
	setPixel(img, ox+68, oy+46, black)

	// Blush marks
	fillEllipse(img, ox+43, oy+42, 5, 3, blush)
	fillEllipse(img, ox+85, oy+42, 5, 3, blush)

	// Arms extending forward over the table
	// Left arm — big vertical difference between raised and hitting
	leftArmY := oy + 78 // raised high
	if leftDown {
		leftArmY = oy + 96 // slammed onto table
	}
	drawArm(img, ox+38, oy+72, ox+20, leftArmY, leftDown)

	// Right arm
	rightArmY := oy + 78
	if rightDown {
		rightArmY = oy + 96
	}
	drawArm(img, ox+90, oy+72, ox+108, rightArmY, rightDown)

	// Impact lines when hitting
	if leftDown {
		drawImpact(img, ox+24, leftArmY-6)
	}
	if rightDown {
		drawImpact(img, ox+104, rightArmY-6)
	}
}

func drawArm(img *image.NRGBA, sx, sy, ex, ey int, hitting bool) {
	// Draw a tapered arm from shoulder to paw
	dx := float64(ex - sx)
	dy := float64(ey - sy)
	length := math.Sqrt(dx*dx + dy*dy)
	steps := int(length)

	for i := range steps {
		t := float64(i) / float64(steps)
		x := float64(sx) + dx*t
		y := float64(sy) + dy*t
		// Taper from 8px at shoulder to 6px at paw
		r := 4.0 - t*1.0
		fillEllipseF(img, x, y, r, r, white)
	}
	// Outline the arm edges
	for i := range steps {
		t := float64(i) / float64(steps)
		x := float64(sx) + dx*t
		y := float64(sy) + dy*t
		r := 4.0 - t*1.0
		drawOutlineF(img, x, y, r, r, black, 2.2)
	}
	// Arm fill again on top of outline to clean up interior
	for i := range steps {
		t := float64(i) / float64(steps)
		x := float64(sx) + dx*t
		y := float64(sy) + dy*t
		r := 3.5 - t*1.0
		fillEllipseF(img, x, y, r, r, white)
	}

	// Paw at the end
	pawRX := 10.0
	pawRY := 6.0
	if hitting {
		pawRY = 4.0 // squished
		pawRX = 12.0
	}
	fillEllipseF(img, float64(ex), float64(ey), pawRX, pawRY, white)
	drawOutlineF(img, float64(ex), float64(ey), pawRX, pawRY, black, 2.0)
	fillEllipseF(img, float64(ex), float64(ey), pawRX-2, pawRY-1, white) // clean interior

	// Paw pads
	fillEllipseF(img, float64(ex)-3, float64(ey), 2.5, 1.5, padPink)
	fillEllipseF(img, float64(ex)+3, float64(ey), 2.5, 1.5, padPink)
	fillEllipseF(img, float64(ex), float64(ey)+1, 3, 2, padPink)
}

func drawImpact(img *image.NRGBA, cx, cy int) {
	// Small lines radiating outward
	for _, angle := range []float64{-0.8, -0.4, 0, 0.4, 0.8} {
		for r := 4.0; r < 9.0; r += 1.0 {
			x := float64(cx) + r*math.Sin(angle)
			y := float64(cy) - r*math.Cos(angle)
			setPixel(img, int(x), int(y), black)
		}
	}
}

func drawFilledEar(img *image.NRGBA, cx, cy int, isLeft bool) {
	dir := 1
	if isLeft {
		dir = -1
	}
	// Outer ear (white with black outline)
	for dy := 0; dy < 20; dy++ {
		w := dy * 9 / 20
		for dx := -w; dx <= w; dx++ {
			x := cx + dx*dir
			setPixel(img, x, cy+dy, white)
		}
		// Outline edges
		setPixel(img, cx+(-w-1)*dir, cy+dy, black)
		setPixel(img, cx+(w+1)*dir, cy+dy, black)
	}
	// Tip outline
	setPixel(img, cx, cy, black)
	setPixel(img, cx-dir, cy+1, black)
	setPixel(img, cx+dir, cy+1, black)

	// Inner ear (pink)
	for dy := 4; dy < 18; dy++ {
		w := (dy - 2) * 5 / 20
		for dx := -w; dx <= w; dx++ {
			x := cx + dx*dir
			setPixel(img, x, cy+dy, pink)
		}
	}
}

// --- Drawing primitives ---

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	b := img.Bounds()
	if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
		// Alpha blend
		existing := img.NRGBAAt(x, y)
		if c.A == 255 || existing.A == 0 {
			img.SetNRGBA(x, y, c)
		} else if c.A > 0 {
			a := float64(c.A) / 255.0
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(float64(c.R)*a + float64(existing.R)*(1-a)),
				G: uint8(float64(c.G)*a + float64(existing.G)*(1-a)),
				B: uint8(float64(c.B)*a + float64(existing.B)*(1-a)),
				A: max(existing.A, c.A),
			})
		}
	}
}

func fillRect(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			setPixel(img, x, y, c)
		}
	}
}

func fillEllipse(img *image.NRGBA, cx, cy, rx, ry int, c color.NRGBA) {
	fillEllipseF(img, float64(cx), float64(cy), float64(rx), float64(ry), c)
}

func fillEllipseF(img *image.NRGBA, cx, cy, rx, ry float64, c color.NRGBA) {
	for dy := int(-ry - 1); dy <= int(ry+1); dy++ {
		for dx := int(-rx - 1); dx <= int(rx+1); dx++ {
			d := float64(dx*dx)/(rx*rx) + float64(dy*dy)/(ry*ry)
			if d <= 1.0 {
				setPixel(img, int(cx)+dx, int(cy)+dy, c)
			}
		}
	}
}

func drawOutline(img *image.NRGBA, cx, cy, rx, ry int, c color.NRGBA, thickness float64) {
	drawOutlineF(img, float64(cx), float64(cy), float64(rx), float64(ry), c, thickness)
}

func drawOutlineF(img *image.NRGBA, cx, cy, rx, ry float64, c color.NRGBA, thickness float64) {
	for dy := int(-ry - 2); dy <= int(ry+2); dy++ {
		for dx := int(-rx - 2); dx <= int(rx+2); dx++ {
			d := float64(dx*dx)/(rx*rx) + float64(dy*dy)/(ry*ry)
			if d >= 0.8 && d <= 1.0+thickness*0.1 {
				setPixel(img, int(cx)+dx, int(cy)+dy, c)
			}
		}
	}
}
