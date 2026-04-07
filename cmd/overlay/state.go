package main

import "time"

type animState int

const (
	stateIdle animState = iota
	stateLeftPaw
	stateRightPaw
	stateBothPaws
)

func (s animState) String() string {
	switch s {
	case stateIdle:
		return "idle"
	case stateLeftPaw:
		return "left_paw"
	case stateRightPaw:
		return "right_paw"
	case stateBothPaws:
		return "both_paws"
	default:
		return "unknown"
	}
}

type state struct {
	st          animState
	nextPaw     animState
	lastInput   time.Time
	idleTimeout time.Duration
	leftHeld    bool
	rightHeld   bool
}

func newState() *state {
	return &state{
		st:          stateIdle,
		nextPaw:     stateLeftPaw,
		idleTimeout: 200 * time.Millisecond,
	}
}

func (s *state) current() animState {
	st := s.st
	s.leftHeld = false
	s.rightHeld = false
	return st
}

func (s *state) onKey() {
	s.lastInput = time.Now()
	s.st = s.nextPaw
	if s.nextPaw == stateLeftPaw {
		s.nextPaw = stateRightPaw
	} else {
		s.nextPaw = stateLeftPaw
	}
}

func (s *state) onLeftClick() {
	s.lastInput = time.Now()
	s.leftHeld = true
	if s.rightHeld {
		s.st = stateBothPaws
	} else {
		s.st = stateLeftPaw
	}
}

func (s *state) onRightClick() {
	s.lastInput = time.Now()
	s.rightHeld = true
	if s.leftHeld {
		s.st = stateBothPaws
	} else {
		s.st = stateRightPaw
	}
}

func (s *state) tick() {
	if s.st != stateIdle && time.Since(s.lastInput) >= s.idleTimeout {
		s.st = stateIdle
		s.leftHeld = false
		s.rightHeld = false
	}
}
