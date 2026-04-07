package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

const (
	saveInterval = 10 * time.Second
	waybarSignal = 10
)

var dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "bongocat")

func stateFile() string    { return filepath.Join(dataDir, "state.json") }
func lifetimeFile() string { return filepath.Join(dataDir, "lifetime.txt") }
func modeFile() string     { return filepath.Join(dataDir, "mode.txt") }

type lifetimeData struct {
	Keys  int64 `json:"keys"`
	Left  int64 `json:"left"`
	Right int64 `json:"right"`
}

type stateJSON struct {
	Text     string `json:"text"`
	Mode     string `json:"mode"`
	Session  int64  `json:"session"`
	Lifetime int64  `json:"lifetime"`
}

func formatCount(n int64) string {
	if n < 0 {
		return "-" + formatCount(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
		b.WriteByte(',')
	}
	for i := remainder; i < len(s); i += 3 {
		if i > remainder {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func loadLifetime() lifetimeData {
	data, err := os.ReadFile(lifetimeFile())
	if err != nil {
		return lifetimeData{}
	}
	var ld lifetimeData
	if err := json.Unmarshal(data, &ld); err != nil {
		return lifetimeData{}
	}
	return ld
}

func saveLifetime(ld lifetimeData) {
	data, _ := json.Marshal(ld)
	tmp := lifetimeFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	os.Rename(tmp, lifetimeFile())
}

func loadMode() string {
	data, err := os.ReadFile(modeFile())
	if err != nil {
		return "session"
	}
	mode := strings.TrimSpace(string(data))
	if mode == "session" || mode == "lifetime" {
		return mode
	}
	return "session"
}

func saveMode(mode string) {
	os.WriteFile(modeFile(), []byte(mode), 0644)
}

func writeState(mode string, session int64, lt lifetimeData, leftClicks, rightClicks int64) {
	fc := formatCount
	keyWidth := max(len(fc(session)), len(fc(lt.Keys)))
	clickWidth := max(len(fc(leftClicks)), len(fc(lt.Left)))
	rclickWidth := max(len(fc(rightClicks)), len(fc(lt.Right)))

	var text string
	if mode == "session" {
		text = fmt.Sprintf("\u23f3 %*s \U0001f5b1 %*s|%*s",
			keyWidth, fc(session),
			clickWidth, fc(leftClicks),
			rclickWidth, fc(rightClicks))
	} else {
		text = fmt.Sprintf("\u267e\ufe0f %*s \U0001f5b1 %*s|%*s",
			keyWidth, fc(lt.Keys),
			clickWidth, fc(lt.Left),
			rclickWidth, fc(lt.Right))
	}

	state := stateJSON{
		Text:     text,
		Mode:     mode,
		Session:  session,
		Lifetime: lt.Keys,
	}
	data, _ := json.Marshal(state)
	tmp := stateFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	os.Rename(tmp, stateFile())
}

// signalWaybar sends SIGRTMIN+10 to all waybar processes, causing them
// to re-read the custom module's output file. This is the mechanism
// waybar uses for on-demand refresh without polling.
func signalWaybar() {
	exec.Command("pkill", fmt.Sprintf("-RTMIN+%d", waybarSignal), "waybar").Run()
}

func main() {
	log.SetFlags(0)
	os.MkdirAll(dataDir, 0755)

	keyboards, mice := detectDevices()
	if len(keyboards) == 0 {
		log.Fatal("No keyboard devices detected")
	}
	if len(mice) == 0 {
		log.Fatal("No mouse devices detected")
	}
	for _, d := range keyboards {
		log.Printf("keyboard: %s (%s)", d.name, d.path)
	}
	for _, d := range mice {
		log.Printf("mouse:    %s (%s)", d.name, d.path)
	}

	lt := loadLifetime()
	mode := loadMode()
	var session, leftClicks, rightClicks int64
	lastSave := time.Now()

	writeState(mode, session, lt, leftClicks, rightClicks)
	signalWaybar()

	events := make(chan keyEvent, 256)

	var devices []*os.File
	for _, d := range slices.Concat(keyboards, mice) {
		f, err := openDevice(d.path)
		if err != nil {
			log.Printf("warning: cannot open %s: %v", d.path, err)
			continue
		}
		devices = append(devices, f)
		go readDevice(f, events)
	}
	defer func() {
		for _, f := range devices {
			f.Close()
		}
	}()

	if len(devices) == 0 {
		log.Fatal("No devices could be opened")
	}

	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, syscall.SIGTERM, syscall.SIGINT)

	sigToggle := make(chan os.Signal, 1)
	signal.Notify(sigToggle, syscall.SIGUSR1)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	cleanup := func() {
		saveLifetime(lt)
		writeState(mode, session, lt, leftClicks, rightClicks)
		signalWaybar()
	}

	for {
		select {
		case <-sigStop:
			cleanup()
			return

		case <-sigToggle:
			if mode == "session" {
				mode = "lifetime"
			} else {
				mode = "session"
			}
			saveMode(mode)
			writeState(mode, session, lt, leftClicks, rightClicks)
			signalWaybar()

		case ev := <-events:
			switch ev.code {
			case btnLeft:
				leftClicks++
				lt.Left++
			case btnRight:
				rightClicks++
				lt.Right++
			default:
				session++
				lt.Keys++
			}
			writeState(mode, session, lt, leftClicks, rightClicks)
			signalWaybar()

		case <-ticker.C:
			if time.Since(lastSave) >= saveInterval {
				saveLifetime(lt)
				lastSave = time.Now()
			}
		}
	}
}
