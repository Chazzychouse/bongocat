package evdev

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unsafe"
)

// Evdev event types and codes from <linux/input-event-codes.h>.
// These are stable kernel ABI — values won't change across versions.
const (
	evKey   = 0x01 // EV_KEY: key press/release events (keyboards + mouse buttons)
	evRel   = 0x02 // EV_REL: relative axis movement (mice, trackballs)
	keyDown = 1    // Key/button pressed (as opposed to 0=released, 2=repeat)

	keyA = 30   // KEY_A: used to detect if a device has alphabetic keys (i.e. is a keyboard)
	relX = 0x00 // REL_X: horizontal mouse movement axis
	relY = 0x01 // REL_Y: vertical mouse movement axis

	BtnLeft  = 0x110 // BTN_LEFT: left mouse button
	BtnRight = 0x111 // BTN_RIGHT: right mouse button
)

// inputEventSize is the size of the Linux input_event struct (16 bytes time + 2+2+4 data).
//
//	struct input_event {
//	    struct timeval time; // Event timestamp
//	    __u16 type;          // Event type (e.g., EV_KEY, EV_REL)
//	    __u16 code;          // Event code (specific key or axis)
//	    __s32 value;         // Event value (state or amount of change)
//	};
const inputEventSize = 24

// evIoc builds a Linux ioctl request number from its components,
// matching the kernel's _IOC macro from <asm-generic/ioctl.h>:
//
//	dir  = _IOC_READ (2) or _IOC_WRITE (1)
//	type = device type character ('E' = 0x45 for evdev)
//	nr   = command number within the type
//	size = expected payload size in bytes
func evIoc(dir, typ, nr, size uintptr) uintptr {
	return (dir << 30) | (size << 16) | (typ << 8) | nr
}

// eviocgname returns the ioctl number for EVIOCGNAME(length), which
// reads the human-readable device name (e.g. "AT Translated Set 2 keyboard").
func eviocgname(length uintptr) uintptr { return evIoc(2, 'E', 0x06, length) }

// eviocgbit returns the ioctl number for EVIOCGBIT(ev, length), which
// reads the capability bitmask for a given event type. For example,
// EVIOCGBIT(EV_KEY, size) returns which key codes the device can produce.
func eviocgbit(ev, length uintptr) uintptr { return evIoc(2, 'E', 0x20+ev, length) }

// evdevName queries the kernel for the human-readable name of an evdev
// device. Returns "" if the ioctl fails (e.g. insufficient permissions).
func evdevName(fd uintptr) string {
	buf := make([]byte, 256)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgname(256), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return ""
	}
	// The kernel null-terminates the name inside the buffer.
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

// evdevBits queries which codes a device supports for a given event type.
// It returns a bitmask where bit N being set means the device can produce
// events with that code. For example, evdevBits(fd, EV_KEY, 768) returns
// which key codes are supported. Returns nil if the ioctl fails.
func evdevBits(fd uintptr, evType, maxBit int) []byte {
	size := (maxBit + 7) / 8
	bits := make([]byte, size)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgbit(uintptr(evType), uintptr(size)), uintptr(unsafe.Pointer(&bits[0])))
	if errno != 0 {
		return nil
	}
	return bits
}

// hasBit checks whether a specific bit is set in a bitmask returned by evdevBits.
func hasBit(bits []byte, bit int) bool {
	idx := bit / 8
	if idx >= len(bits) {
		return false
	}
	return bits[idx]&(1<<uint(bit%8)) != 0
}

// InputDevice pairs an evdev node path (e.g. "/dev/input/event3") with
// the human-readable device name reported by the kernel.
type InputDevice struct {
	Path string
	Name string
}

// probeDevice opens an evdev node and classifies it as keyboard, mouse, or
// neither by querying its capability bitmasks via ioctl:
//   - Keyboard: supports EV_KEY and can produce KEY_A (filters out power
//     buttons, volume knobs, etc. that also report EV_KEY).
//   - Mouse: supports EV_REL with both X/Y axes and has BTN_LEFT (filters
//     out accelerometers and other relative-axis devices).
//
// A single device can be classified as both (e.g. some gaming mice with
// macro keys), in which case it appears in both lists.
func probeDevice(path string) (isKb, isMouse bool, name string) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false, false, ""
	}
	defer f.Close()

	fd := f.Fd()
	name = evdevName(fd)

	// Query top-level event type bitmask (which EV_* types this device supports).
	evBits := evdevBits(fd, 0, 32)
	if evBits == nil {
		return false, false, name
	}

	if hasBit(evBits, evKey) {
		if keyBits := evdevBits(fd, evKey, 768); keyBits != nil {
			isKb = hasBit(keyBits, keyA)
		}
	}

	if hasBit(evBits, evRel) {
		relBits := evdevBits(fd, evRel, 64)
		keyBits := evdevBits(fd, evKey, 768)
		if relBits != nil && keyBits != nil &&
			hasBit(relBits, relX) && hasBit(relBits, relY) &&
			hasBit(keyBits, BtnLeft) {
			isMouse = true
		}
	}

	return isKb, isMouse, name
}

// DetectDevices scans /dev/input/event* and classifies each device as
// keyboard (has EV_KEY with alphabetic keys) or mouse (has EV_REL with
// X/Y axes and BTN_LEFT). If a Makima virtual keyboard is found it is
// preferred exclusively, since Makima aggregates all physical keyboard
// input and listening on both would double-count.
func DetectDevices() (keyboards, mice []InputDevice) {
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, path := range paths {
		isKb, isMouse, name := probeDevice(path)
		dev := InputDevice{Path: path, Name: name}
		if isKb {
			keyboards = append(keyboards, dev)
		}
		if isMouse {
			mice = append(mice, dev)
		}
	}

	// Prefer Makima if present — it aggregates all physical keyboard input.
	if makima := slices.DeleteFunc(slices.Clone(keyboards), func(d InputDevice) bool {
		return !strings.Contains(strings.ToLower(d.Name), "makima")
	}); len(makima) > 0 {
		keyboards = makima
	}

	return
}

// KeyEvent carries a key/button code from reader goroutines to the
// main select loop. Only key-down events are sent; repeats and
// releases are filtered out at the source.
type KeyEvent struct {
	Code uint16
}

// OpenDevice opens an evdev node in read-only mode for event streaming.
func OpenDevice(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

// ReadDevice reads input events from f and sends key-down events on ch.
// Each device gets its own goroutine doing blocking reads. The kernel
// delivers events in batches, so we read up to 64 events at a time and
// parse only the fields we need (type, code, value), skipping the
// 16-byte timestamp we don't use.
func ReadDevice(f *os.File, ch chan<- KeyEvent) {
	buf := make([]byte, inputEventSize*64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return
		}
		for off := 0; off+inputEventSize <= n; off += inputEventSize {
			typ := binary.LittleEndian.Uint16(buf[off+16:])
			code := binary.LittleEndian.Uint16(buf[off+18:])
			value := int32(binary.LittleEndian.Uint32(buf[off+20:]))

			if typ == evKey && value == keyDown {
				ch <- KeyEvent{Code: code}
			}
		}
	}
}
