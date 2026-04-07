package main

import "time"

const maxHistory = 5

type keyEntry struct {
	name    string
	isMouse bool
	time    time.Time
}

type state struct {
	history     []keyEntry
	lastInput   time.Time
	idleTimeout time.Duration
}

func newState() *state {
	return &state{
		idleTimeout: 200 * time.Millisecond,
	}
}

func (s *state) push(name string, isMouse bool) {
	s.lastInput = time.Now()
	entry := keyEntry{name: name, isMouse: isMouse, time: s.lastInput}
	s.history = append([]keyEntry{entry}, s.history...)
	if len(s.history) > maxHistory {
		s.history = s.history[:maxHistory]
	}
}

func (s *state) tick() bool {
	if len(s.history) == 0 {
		return false
	}
	if time.Since(s.lastInput) >= s.idleTimeout {
		s.history = s.history[:0]
		return true
	}
	return false
}

func (s *state) entries() []keyEntry {
	return s.history
}

func (s *state) active() bool {
	return len(s.history) > 0
}
