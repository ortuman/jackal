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
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	utilstring "github.com/ortuman/jackal/util/string"
	"github.com/ortuman/jackal/xmpp"
	"golang.org/x/crypto/pbkdf2"
)

// ScramType represents a scram autheticator class
type ScramType int

const (
	// ScramSHA1 represents SCRAM-SHA-1 authentication method.
	ScramSHA1 ScramType = iota

	// ScramSHA256 represents SCRAM-SHA-256 authentication method.
	ScramSHA256
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

// Scram represents a SCRAM authenticator.
type Scram struct {
	authenticated bool
	usesCb        bool
	stm           stream.C2S
	userRep       repository.User
	tr            transport.Transport
	tp            ScramType
	h             func() hash.Hash
	state         scramState
	params        *scramParameters
	user          *model.User
	srvNonce      string
	firstMessage  string
}

// NewScram returns a new scram authenticator instance.
func NewScram(stm stream.C2S, tr transport.Transport, scramType ScramType, usesChannelBinding bool, userRep repository.User) *Scram {
	s := &Scram{
		stm:     stm,
		userRep: userRep,
		tr:      tr,
		tp:      scramType,
		usesCb:  usesChannelBinding,
		state:   startScramState,
	}
	switch s.tp {
	case ScramSHA1:
		s.h = sha1.New
	case ScramSHA256:
		s.h = sha256.New
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
	s.user, err = s.userRep.FetchUser(ctx, username)
	switch {
	case err != nil:
		return err
	case s.user == nil:
		return ErrSASLNotAuthorized
	}

	s.srvNonce = cNonce + "-" + uuid.New().String()
	sb64 := base64.StdEncoding.EncodeToString(s.user.Salt)
	s.firstMessage = fmt.Sprintf("r=%s,s=%s,i=%d", s.srvNonce, sb64, s.user.IterationCount)

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

	var saltedPassword []byte
	switch s.tp {
	case ScramSHA1:
		saltedPassword = s.user.PasswordScramSHA1
	case ScramSHA256:
		saltedPassword = s.user.PasswordScramSHA256
	default:
		// This should never be reached, if it does it indicates that a serious bug
		// was introduced somewhere, so make sure we get a report about it instead
		// of just failing auth.
		panic("invalid auth mechanism used")
	}

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
	if subtle.ConstantTimeCompare([]byte(clientFinalMessage), []byte(p)) != 1 {
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

	// https://tools.ietf.org/html/rfc5801#section-5
	switch gs2BindFlag {
	case "p":
		// Channel binding is supported and required.
		if !s.usesCb {
			return ErrSASLNotAuthorized
		}
	case "n", "y":
		// Channel binding is not supported, or is supported but is not required.
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
		key, val := utilstring.SplitKeyAndValue(authzID, '=')
		if len(key) == 0 || key != "a" {
			return ErrSASLMalformedRequest
		}
		p.authzID = val
	}
	for i := 2; i < len(sp); i++ {
		key, val := utilstring.SplitKeyAndValue(sp[i], '=')
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

// SaltedPassword computes a salted password using the HMAC variant of PBKDF2.
//
// For OWASP recommendations for tuning PBKDF2 see:
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
//
// For NIST recommendations, see: the legacy SP 800-132 and SP 80063b §5.1.1.2
// Memorized Secret Verifiers.
func SaltedPassword(password, salt []byte, iterationCount int, h func() hash.Hash) []byte {
	hKeyLen := h().Size()
	return pbkdf2.Key(password, salt, iterationCount, hKeyLen, h)
}
