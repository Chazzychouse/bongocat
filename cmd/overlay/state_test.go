package main

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := newState()
	if s.current() != stateIdle {
		t.Errorf("initial state = %v, want idle", s.current())
	}
}

func TestKeyAlternation(t *testing.T) {
	s := newState()

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Errorf("first key = %v, want left_paw", s.current())
	}

	s.onKey()
	if s.current() != stateRightPaw {
		t.Errorf("second key = %v, want right_paw", s.current())
	}

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Errorf("third key = %v, want left_paw", s.current())
	}
}

func TestMouseButtons(t *testing.T) {
	s := newState()

	s.onLeftClick()
	if s.current() != stateLeftPaw {
		t.Errorf("left click = %v, want left_paw", s.current())
	}

	// Reset to idle before testing right click independently
	s2 := newState()
	s2.onRightClick()
	if s2.current() != stateRightPaw {
		t.Errorf("right click = %v, want right_paw", s2.current())
	}
}

func TestBothPaws(t *testing.T) {
	s := newState()

	s.onLeftClick()
	s.onRightClick()
	if s.current() != stateBothPaws {
		t.Errorf("left+right = %v, want both_paws", s.current())
	}
}

func TestIdleTimeout(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond

	s.onKey()
	if s.current() != stateLeftPaw {
		t.Fatalf("after key = %v, want left_paw", s.current())
	}

	time.Sleep(100 * time.Millisecond)
	s.tick()
	if s.current() != stateIdle {
		t.Errorf("after timeout = %v, want idle", s.current())
	}
}

func TestNoIdleWhileActive(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond

	s.onKey()
	time.Sleep(30 * time.Millisecond)
	s.onKey()
	time.Sleep(30 * time.Millisecond)
	s.tick()
	if s.current() == stateIdle {
		t.Error("should not be idle — input was recent")
	}
}
