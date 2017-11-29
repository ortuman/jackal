/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/entity"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
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

type scramParameter struct {
	key string
	val string
}

type scramParameters struct {
	gs2Header   string
	cbMechanism string
	authzID     string
	params      []scramParameter
}

func (s *scramParameters) getParameter(key string) string {
	for _, p := range s.params {
		if p.key == key {
			return p.val
		}
	}
	return ""
}

func (s *scramParameters) String() string {
	ret := ""
	for i, p := range s.params {
		if i != 0 {
			ret += ","
		}
		ret += fmt.Sprintf("%s=%s", p.key, p.val)
	}
	return ret
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
	username := s.params.getParameter("n")
	cNonce := s.params.getParameter("r")

	if len(username) == 0 || len(cNonce) == 0 {
		return errSASLMalformedRequest
	}
	user, err := storage.Instance().FetchUser(username)
	if err != nil {
		return err
	}
	if user == nil {
		return errSASLNotAuthorized
	}
	s.user = user

	s.srvNonce = cNonce + "-" + uuid.New()
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
	c := s.getCBindInputString()
	initialMessage := s.params.String()
	clientFinalMessageBare := fmt.Sprintf("c=%s,r=%s", c, s.srvNonce)

	var h func() hash.Hash
	var keyLen int
	if s.tp == sha1ScramType {
		h = sha1.New
		keyLen = sha1.Size
	} else {
		h = sha256.New
		keyLen = sha256.Size
	}
	saltedPassword := pbkdf2.Key([]byte(s.user.Password), s.salt, iterationsCount, keyLen, h)
	clientKey := mac([]byte("Client Key"), saltedPassword, h)

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
		p.params = append(p.params, scramParameter{key, val})
	}
	s.params = p
	return nil
}

func (s *scramAuthenticator) getCBindInputString() string {
	buf := new(bytes.Buffer)
	buf.Write([]byte(s.params.gs2Header))
	if s.usesCb {
		buf.Write(s.strm.ChannelBindingBytes(s.params.cbMechanism))
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func mac(b []byte, key []byte, h func() hash.Hash) []byte {
	m := hmac.New(h, key)
	m.Write(b)
	return m.Sum(nil)
}
