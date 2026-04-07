package main

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := newState()
	if s.active() {
		t.Error("new state should not be active")
	}
	if len(s.entries()) != 0 {
		t.Errorf("entries = %d, want 0", len(s.entries()))
	}
}

func TestPushAddsEntries(t *testing.T) {
	s := newState()
	s.push("A", false)
	entries := s.entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].name != "A" {
		t.Errorf("name = %q, want A", entries[0].name)
	}
	if entries[0].isMouse {
		t.Error("isMouse should be false")
	}
}

func TestPushNewestFirst(t *testing.T) {
	s := newState()
	s.push("A", false)
	s.push("B", false)
	s.push("C", false)
	entries := s.entries()
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if entries[0].name != "C" {
		t.Errorf("entries[0] = %q, want C", entries[0].name)
	}
	if entries[1].name != "B" {
		t.Errorf("entries[1] = %q, want B", entries[1].name)
	}
	if entries[2].name != "A" {
		t.Errorf("entries[2] = %q, want A", entries[2].name)
	}
}

func TestPushCapsAtFive(t *testing.T) {
	s := newState()
	for _, k := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		s.push(k, false)
	}
	entries := s.entries()
	if len(entries) != 5 {
		t.Fatalf("entries = %d, want 5", len(entries))
	}
	if entries[0].name != "G" {
		t.Errorf("newest = %q, want G", entries[0].name)
	}
	if entries[4].name != "C" {
		t.Errorf("oldest = %q, want C", entries[4].name)
	}
}

func TestPushMouse(t *testing.T) {
	s := newState()
	s.push("LMB", true)
	entries := s.entries()
	if !entries[0].isMouse {
		t.Error("isMouse should be true")
	}
}

func TestActive(t *testing.T) {
	s := newState()
	if s.active() {
		t.Error("should not be active when empty")
	}
	s.push("A", false)
	if !s.active() {
		t.Error("should be active after push")
	}
}

func TestTickIdleTimeout(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond
	s.push("A", false)

	time.Sleep(100 * time.Millisecond)
	changed := s.tick()
	if !changed {
		t.Error("tick should return true when clearing")
	}
	if s.active() {
		t.Error("should be idle after timeout")
	}
	if len(s.entries()) != 0 {
		t.Errorf("entries = %d, want 0", len(s.entries()))
	}
}

func TestTickNoChangeWhileActive(t *testing.T) {
	s := newState()
	s.idleTimeout = 50 * time.Millisecond
	s.push("A", false)

	time.Sleep(30 * time.Millisecond)
	s.push("B", false)
	time.Sleep(30 * time.Millisecond)
	changed := s.tick()
	if changed {
		t.Error("tick should return false — input was recent")
	}
	if !s.active() {
		t.Error("should still be active")
	}
}

func TestTickNoChangeWhenAlreadyIdle(t *testing.T) {
	s := newState()
	changed := s.tick()
	if changed {
		t.Error("tick on empty state should return false")
	}
}
