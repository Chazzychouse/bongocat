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

const (
	evKey   = 0x01
	evRel   = 0x02
	keyDown = 1

	keyA = 30
	relX = 0x00
	relY = 0x01

	BtnLeft  = 0x110
	BtnRight = 0x111
)

const inputEventSize = 24

func evIoc(dir, typ, nr, size uintptr) uintptr {
	return (dir << 30) | (size << 16) | (typ << 8) | nr
}

func eviocgname(length uintptr) uintptr { return evIoc(2, 'E', 0x06, length) }
func eviocgbit(ev, length uintptr) uintptr { return evIoc(2, 'E', 0x20+ev, length) }

func evdevName(fd uintptr) string {
	buf := make([]byte, 256)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgname(256), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func evdevBits(fd uintptr, evType, maxBit int) []byte {
	size := (maxBit + 7) / 8
	bits := make([]byte, size)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, eviocgbit(uintptr(evType), uintptr(size)), uintptr(unsafe.Pointer(&bits[0])))
	if errno != 0 {
		return nil
	}
	return bits
}

func hasBit(bits []byte, bit int) bool {
	idx := bit / 8
	if idx >= len(bits) {
		return false
	}
	return bits[idx]&(1<<uint(bit%8)) != 0
}

type InputDevice struct {
	Path string
	Name string
}

func probeDevice(path string) (isKb, isMouse bool, name string) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false, false, ""
	}
	defer f.Close()

	fd := f.Fd()
	name = evdevName(fd)

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

	if makima := slices.DeleteFunc(slices.Clone(keyboards), func(d InputDevice) bool {
		return !strings.Contains(strings.ToLower(d.Name), "makima")
	}); len(makima) > 0 {
		keyboards = makima
	}

	return
}

type KeyEvent struct {
	Code uint16
}

func OpenDevice(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

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
