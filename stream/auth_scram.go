/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/entity"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const iterationsCount = 4096

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
	username    string
	cNonce      string
}

type scramAuthenticator struct {
	strm          *Stream
	tp            scramType
	usesCb        bool
	state         scramState
	params        *scramParameters
	user          *entity.User
	salt          []byte
	srvNonce      string
	firstMessage  string
	authenticated bool
}

func newScram(strm *Stream, scramType scramType, usesChannelBinding bool) authenticator {
	s := &scramAuthenticator{
		strm:   strm,
		tp:     scramType,
		usesCb: usesChannelBinding,
		state:  startScramState,
	}
	return s
}

func (s *scramAuthenticator) Mechanism() string {
	switch s.tp {
	case sha1ScramType:
		if s.usesCb {
			return "SCRAM-SHA-1-PLUS"
		}
		return "SCRAM-SHA-1"

	case sha256ScramType:
		if s.usesCb {
			return "SCRAM-SHA-256-PLUS"
		}
		return "SCRAM-SHA-256"
	}
	return ""
}

func (s *scramAuthenticator) Username() string {
	if s.authenticated {
		return s.user.Username
	}
	return ""
}

func (s *scramAuthenticator) Authenticated() bool {
	return s.authenticated
}

func (s *scramAuthenticator) UsesChannelBinding() bool {
	return s.usesCb
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
	s.authenticated = false

	s.state = startScramState
	s.params = nil
	s.user = nil
	s.salt = nil
	s.srvNonce = ""
	s.firstMessage = ""
}

func (s *scramAuthenticator) handleStart(elem *xml.Element) error {
	p, err := s.getElementPayload(elem)
	if err != nil {
		return err
	}
	if err := s.parseParameters(p); err != nil {
		return err
	}

	user, err := storage.Instance().FetchUser(s.params.username)
	if err != nil {
		return err
	}
	if user == nil {
		return errSASLNotAuthorized
	}
	s.user = user

	s.srvNonce = s.params.cNonce + "-" + uuid.New()
	s.salt = util.RandomBytes(32)
	sb64 := base64.StdEncoding.EncodeToString(s.salt)
	s.firstMessage = fmt.Sprintf("r=%s,s=%s,i=%d", s.srvNonce, sb64, iterationsCount)

	respElem := xml.NewMutableElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(s.firstMessage)))
	s.strm.SendElement(respElem.Copy())

	s.state = challengedScramState
	return nil
}

func (s *scramAuthenticator) handleChallenged(elem *xml.Element) error {
	p, err := s.getElementPayload(elem)
	if err != nil {
		return err
	}
	println(p)
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

func (s *scramAuthenticator) parseParameters(str string) error {
	p := &scramParameters{}

	sp := strings.Split(str, ",")
	if len(sp) < 2 {
		return errSASLIncorrectEncoding
	}
	gs2BindFlag := sp[0]

	switch gs2BindFlag {
	case "y":
		if !s.usesCb {
			return errSASLNotAuthorized
		}
	case "n":
		break
	default:
		if !strings.HasPrefix(gs2BindFlag, "p=") {
			return errSASLMalformedRequest
		}
		if !s.usesCb {
			return errSASLNotAuthorized
		}
		p.cbMechanism = gs2BindFlag[2:]
	}
	authzID := sp[1]
	p.gs2Header = gs2BindFlag + "," + authzID + ","

	if len(authzID) > 0 {
		key, val := util.SplitKeyAndValue(authzID, '=')
		if len(key) == 0 || key != "a" {
			return errSASLMalformedRequest
		}
		p.authzID = val
	}
	for i := 2; i < len(sp); i++ {
		key, val := util.SplitKeyAndValue(sp[i], '=')
		switch key {
		case "r":
			p.cNonce = val
		case "n":
			p.username = val
		default:
			break
		}
	}
	if len(p.username) == 0 || len(p.cNonce) == 0 {
		return errSASLMalformedRequest
	}
	s.params = p
	return nil
}
