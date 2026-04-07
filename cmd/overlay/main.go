package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/chazzy/bongocat/internal/evdev"
)

var animDir = filepath.Join(os.Getenv("HOME"), ".config", "bongocat", "animations")

func main() {
	log.SetFlags(0)

	animName := flag.String("animation", "bongocat", "animation set name")
	sizeOverride := flag.Int("size", 0, "override window size in pixels")
	flag.Parse()

	mf, err := loadManifest(filepath.Join(animDir, *animName))
	if err != nil {
		log.Fatalf("load manifest: %v", err)
	}

	winSize := mf.FrameWidth
	if *sizeOverride > 0 {
		winSize = *sizeOverride
	}

	keyboards, mice := evdev.DetectDevices()
	if len(keyboards) == 0 && len(mice) == 0 {
		log.Fatal("No input devices detected")
	}
	for _, d := range keyboards {
		log.Printf("keyboard: %s (%s)", d.Name, d.Path)
	}
	for _, d := range mice {
		log.Printf("mouse:    %s (%s)", d.Name, d.Path)
	}

	events := make(chan evdev.KeyEvent, 256)
	var devices []*os.File
	for _, d := range slices.Concat(keyboards, mice) {
		f, err := evdev.OpenDevice(d.Path)
		if err != nil {
			log.Printf("warning: cannot open %s: %v", d.Path, err)
			continue
		}
		devices = append(devices, f)
		go evdev.ReadDevice(f, events)
	}
	defer func() {
		for _, f := range devices {
			f.Close()
		}
	}()

	if len(devices) == 0 {
		log.Fatal("No devices could be opened")
	}

	app := gtk.NewApplication("com.github.bongocat", gio.ApplicationFlagsNone)

	app.ConnectActivate(func() {
		win := gtk.NewApplicationWindow(app)
		win.SetTitle("bongocat")
		win.SetDefaultSize(winSize, winSize)
		win.SetDecorated(false)

		css := gtk.NewCSSProvider()
		css.LoadFromString("window, drawingarea { background-color: transparent; }")
		gtk.StyleContextAddProviderForDisplay(
			gdk.DisplayGetDefault(),
			css,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)

		rend, err := newRenderer(mf)
		if err != nil {
			log.Fatalf("init renderer: %v", err)
		}

		st := newState()

		area := gtk.NewDrawingArea()
		area.SetContentWidth(winSize)
		area.SetContentHeight(winSize)
		area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
			rend.draw(cr, st.current(), w, h)
		})

		win.SetChild(area)

		// SIGUSR2 toggle visibility
		visible := true
		sigToggle := make(chan os.Signal, 1)
		signal.Notify(sigToggle, syscall.SIGUSR2)
		go func() {
			for range sigToggle {
				glib.IdleAdd(func() {
					visible = !visible
					win.SetVisible(visible)
				})
			}
		}()

		// Drain evdev events into state machine on GTK thread
		go func() {
			for ev := range events {
				glib.IdleAdd(func() {
					prev := st.current()
					switch ev.Code {
					case evdev.BtnLeft:
						st.onLeftClick()
					case evdev.BtnRight:
						st.onRightClick()
					default:
						st.onKey()
					}
					if st.current() != prev {
						rend.reset()
					}
					area.QueueDraw()
				})
			}
		}()

		// Animation frame timer + idle check (~60 ticks/sec)
		glib.TimeoutAdd(16, func() bool {
			prev := st.current()
			st.tick()
			if st.current() != prev {
				rend.reset()
				area.QueueDraw()
			} else if st.current() != stateIdle {
				rend.advance()
				area.QueueDraw()
			}
			return true
		})

		win.Show()
		fmt.Printf("overlay: %dx%d animation=%s\n", winSize, winSize, mf.Name)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
