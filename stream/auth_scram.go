/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"encoding/base64"
	"strings"

	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
)

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

type scramParameters struct {
	gs2Header   string
	cbMechanism string
	authzID     string
	n           string
	cNonce      string
}

type scramAuthenticator struct {
	strm               *Stream
	tp                 scramType
	usesChannelBinding bool
	state              scramState
	params             *scramParameters
	authenticated      bool
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
	return s.authenticated
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
	// s.user = nil
	s.authenticated = false

	s.state = startScramState
	s.params = nil
}

func (s *scramAuthenticator) handleStart(elem *xml.Element) error {
	p, err := s.getElementPayload(elem)
	if err != nil {
		return err
	}
	s.parseParameters(p)

	return nil
}

func (s *scramAuthenticator) handleChallenged(elem *xml.Element) error {
	return nil
}

func (s *scramAuthenticator) getElementPayload(elem *xml.Element) (string, error) {
	if elem.TextLen() == 0 {
		return "", errSASLIncorrectEncoding
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return "", errSASLIncorrectEncoding
	}
	return string(b), nil
}

func (s *scramAuthenticator) parseParameters(str string) (*scramParameters, error) {
	p := &scramParameters{}
	sp := strings.Split(str, ",")
	if len(sp) < 2 {
		return nil, errSASLIncorrectEncoding
	}
	gs2BindFlag := sp[0]

	switch gs2BindFlag {
	case "y":
		if !s.usesChannelBinding {
			return nil, errSASLNotAuthorized
		}
	case "n":
		break
	default:
		if !strings.HasPrefix(gs2BindFlag, "p=") {
			return nil, errSASLMalformedRequest
		}
		if !s.usesChannelBinding {
			return nil, errSASLNotAuthorized
		}
		p.cbMechanism = gs2BindFlag[2:]
	}
	authzID := sp[0]
	p.gs2Header = gs2BindFlag + "," + authzID + ","

	if len(authzID) > 0 {
		key, val := util.SplitKeyAndValue(authzID, '=')
		if len(key) == 0 || key != "a" {
			return nil, errSASLMalformedRequest
		}
		p.authzID = val
	}
	for i := 2; i < len(sp); i++ {
		key, val := util.SplitKeyAndValue(authzID, '=')
		switch key {
		case "r":
			p.cNonce = val
		case "n":
			p.n = val
		default:
			return nil, errSASLMalformedRequest
		}
	}
	s.params = p

	return p, nil
}
