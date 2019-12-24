/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xmpp"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
)

// ScramType represents a scram autheticator class
type ScramType int

const (
	// ScramSHA1 represents SCRAM-SHA-1 authentication method.
	ScramSHA1 ScramType = iota

	// ScramSHA256 represents SCRAM-SHA-256 authentication method.
	ScramSHA256

	// ScramSHA512 represents SCRAM-SHA-512 authentication method.
	ScramSHA512
)

const iterationsCount = 4096

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

// Scram represents a SCRAM authenticator.
type Scram struct {
	stm           stream.C2S
	tr            transport.Transport
	tp            ScramType
	usesCb        bool
	h             func() hash.Hash
	hKeyLen       int
	state         scramState
	params        *scramParameters
	user          *model.User
	salt          []byte
	srvNonce      string
	firstMessage  string
	authenticated bool
}

// NewScram returns a new scram authenticator instance.
func NewScram(stm stream.C2S, tr transport.Transport, scramType ScramType, usesChannelBinding bool) *Scram {
	s := &Scram{
		stm:    stm,
		tr:     tr,
		tp:     scramType,
		usesCb: usesChannelBinding,
		state:  startScramState,
	}
	switch s.tp {
	case ScramSHA1:
		s.h = sha1.New
		s.hKeyLen = sha1.Size
	case ScramSHA256:
		s.h = sha256.New
		s.hKeyLen = sha256.Size
	case ScramSHA512:
		s.h = sha512.New
		s.hKeyLen = sha512.Size
	}
	return s
}

// Mechanism returns authenticator mechanism name.
func (s *Scram) Mechanism() string {
	switch s.tp {
	case ScramSHA1:
		if s.usesCb {
			return "SCRAM-SHA-1-PLUS"
		}
		return "SCRAM-SHA-1"

	case ScramSHA256:
		if s.usesCb {
			return "SCRAM-SHA-256-PLUS"
		}
		return "SCRAM-SHA-256"

	case ScramSHA512:
		if s.usesCb {
			return "SCRAM-SHA-512-PLUS"
		}
		return "SCRAM-SHA-512"
	}
	return ""
}

// Username returns authenticated username in case
// authentication process has been completed.
func (s *Scram) Username() string {
	if s.authenticated {
		return s.user.Username
	}
	return ""
}

// Authenticated returns whether or not user has been authenticated.
func (s *Scram) Authenticated() bool {
	return s.authenticated
}

// UsesChannelBinding returns whether or not scram authenticator
// requires channel binding bytes.
func (s *Scram) UsesChannelBinding() bool {
	return s.usesCb
}

// ProcessElement process an incoming authenticator element.
func (s *Scram) ProcessElement(ctx context.Context, elem xmpp.XElement) error {
	if s.Authenticated() {
		return nil
	}
	switch elem.Name() {
	case "auth":
		if s.state == startScramState {
			return s.handleStart(ctx, elem)
		}
	case "response":
		if s.state == challengedScramState {
			return s.handleChallenged(ctx, elem)
		}
	}
	return ErrSASLNotAuthorized
}

// Reset resets scram internal state.
func (s *Scram) Reset() {
	s.authenticated = false

	s.state = startScramState
	s.params = nil
	s.user = nil
	s.salt = nil
	s.srvNonce = ""
	s.firstMessage = ""
}

func (s *Scram) handleStart(ctx context.Context, elem xmpp.XElement) error {
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
		return ErrSASLMalformedRequest
	}
	user, err := storage.FetchUser(username)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrSASLNotAuthorized
	}
	s.user = user

	s.srvNonce = cNonce + "-" + uuid.New()
	s.salt = util.RandomBytes(32)
	sb64 := base64.StdEncoding.EncodeToString(s.salt)
	s.firstMessage = fmt.Sprintf("r=%s,s=%s,i=%d", s.srvNonce, sb64, iterationsCount)

	respElem := xmpp.NewElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(s.firstMessage)))
	s.stm.SendElement(ctx, respElem)

	s.state = challengedScramState
	return nil
}

func (s *Scram) handleChallenged(ctx context.Context, elem xmpp.XElement) error {
	p, err := s.getElementPayload(elem)
	if err != nil {
		return err
	}
	c := s.getCBindInputString()
	initialMessage := s.params.String()
	clientFinalMessageBare := fmt.Sprintf("c=%s,r=%s", c, s.srvNonce)

	saltedPassword := s.pbkdf2([]byte(s.user.Password))
	clientKey := s.hmac([]byte("Client Key"), saltedPassword)
	storedKey := s.hash(clientKey)
	authMessage := initialMessage + "," + s.firstMessage + "," + clientFinalMessageBare
	clientSignature := s.hmac([]byte(authMessage), storedKey)

	clientProof := make([]byte, len(clientKey))
	for i := 0; i < len(clientKey); i++ {
		clientProof[i] = clientKey[i] ^ clientSignature[i]
	}
	serverKey := s.hmac([]byte("Server Key"), saltedPassword)
	serverSignature := s.hmac([]byte(authMessage), serverKey)

	clientFinalMessage := clientFinalMessageBare + ",p=" + base64.StdEncoding.EncodeToString(clientProof)
	if clientFinalMessage != p {
		return ErrSASLNotAuthorized
	}
	v := "v=" + base64.StdEncoding.EncodeToString(serverSignature)

	respElem := xmpp.NewElementNamespace("success", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(v)))
	s.stm.SendElement(ctx, respElem)

	s.authenticated = true
	return nil
}

func (s *Scram) getElementPayload(elem xmpp.XElement) (string, error) {
	if len(elem.Text()) == 0 {
		return "", ErrSASLIncorrectEncoding
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return "", ErrSASLIncorrectEncoding
	}
	return string(b), nil
}

func (s *Scram) parseParameters(str string) error {
	p := &scramParameters{}

	sp := strings.Split(str, ",")
	if len(sp) < 2 {
		return ErrSASLIncorrectEncoding
	}
	gs2BindFlag := sp[0]

	switch gs2BindFlag {
	case "y":
		if !s.usesCb {
			return ErrSASLNotAuthorized
		}
	case "n":
		break
	default:
		if !strings.HasPrefix(gs2BindFlag, "p=") {
			return ErrSASLMalformedRequest
		}
		if !s.usesCb {
			return ErrSASLNotAuthorized
		}
		p.cbMechanism = gs2BindFlag[2:]
	}
	authzID := sp[1]
	p.gs2Header = gs2BindFlag + "," + authzID + ","

	if len(authzID) > 0 {
		key, val := util.SplitKeyAndValue(authzID, '=')
		if len(key) == 0 || key != "a" {
			return ErrSASLMalformedRequest
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

func (s *Scram) getCBindInputString() string {
	buf := new(bytes.Buffer)
	buf.Write([]byte(s.params.gs2Header))
	if s.usesCb {
		switch s.params.cbMechanism {
		case "tls-unique":
			buf.Write(s.tr.ChannelBindingBytes(transport.TLSUnique))
		}
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func (s *Scram) pbkdf2(b []byte) []byte {
	return pbkdf2.Key(b, s.salt, iterationsCount, s.hKeyLen, s.h)
}

func (s *Scram) hmac(b []byte, key []byte) []byte {
	m := hmac.New(s.h, key)
	m.Write(b)
	return m.Sum(nil)
}

func (s *Scram) hash(b []byte) []byte {
	h := s.h()
	h.Write(b)
	return h.Sum(nil)
}
