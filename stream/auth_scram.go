/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/xml"

type scramType int

const (
	sha1ScramType scramType = iota
	sha256ScramType
)

type scramState int

const (
	startScramState scramState = iota
	challengedScramState
)

type scramAuthenticator struct {
	strm               *Stream
	tp                 scramType
	usesChannelBinding bool
	state              scramState
}

func newScram(strm *Stream, scramType scramType, usesChannelBinding bool) authenticator {
	s := &scramAuthenticator{
		strm:               strm,
		tp:                 scramType,
		usesChannelBinding: usesChannelBinding,
		state:              startScramState,
	}
	return s
}

func (s *scramAuthenticator) Mechanism() string {
	switch s.tp {
	case sha1ScramType:
		if s.usesChannelBinding {
			return "SCRAM-SHA-1-PLUS"
		}
		return "SCRAM-SHA-1"

	case sha256ScramType:
		if s.usesChannelBinding {
			return "SCRAM-SHA-256-PLUS"
		}
		return "SCRAM-SHA-256"
	}
	return ""
}

func (s *scramAuthenticator) Username() string {
	return ""
}

func (s *scramAuthenticator) Authenticated() bool {
	return false
}

func (s *scramAuthenticator) UsesChannelBinding() bool {
	return s.usesChannelBinding
}

func (s *scramAuthenticator) ProcessElement(elem *xml.Element) error {
	if s.Authenticated() {
		return nil
	}
	switch elem.Name() {
	case "auth":
		if s.state == startScramState {
			return s.handleStart(elem)
		}
	case "response":
		if s.state == challengedScramState {
			return s.handleChallenged(elem)
		}
	}
	return errSASLNotAuthorized
}

func (s *scramAuthenticator) Reset() {
}

func (s *scramAuthenticator) handleStart(elem *xml.Element) error {
	return nil
}

func (s *scramAuthenticator) handleChallenged(elem *xml.Element) error {
	return nil
}
