// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	usermodel "github.com/ortuman/jackal/pkg/model/user"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/transport"
	stringsutil "github.com/ortuman/jackal/pkg/util/strings"
	"golang.org/x/crypto/sha3"
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

	// ScramSHA3512 represents SCRAM-SHA3-512 authentication method.
	ScramSHA3512
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
	tr            transport.Transport
	tp            ScramType
	usesCb        bool
	rep           repository.User
	peppers       *pepper.Keys
	h             func() hash.Hash
	state         scramState
	params        *scramParameters
	user          *usermodel.User
	srvNonce      string
	firstMessage  string
	authenticated bool
}

// NewScram returns a new scram authenticator instance.
func NewScram(
	tr transport.Transport,
	scramType ScramType,
	usesChannelBinding bool,
	rep repository.User,
	peppers *pepper.Keys,
) *Scram {
	s := &Scram{
		tr:      tr,
		tp:      scramType,
		usesCb:  usesChannelBinding,
		rep:     rep,
		peppers: peppers,
		state:   startScramState,
	}
	switch s.tp {
	case ScramSHA1:
		s.h = sha1.New
	case ScramSHA256:
		s.h = sha256.New
	case ScramSHA512:
		s.h = sha512.New
	case ScramSHA3512:
		s.h = sha3.New512
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

	case ScramSHA3512:
		if s.usesCb {
			return "SCRAM-SHA3-512-PLUS"
		}
		return "SCRAM-SHA3-512"
	}
	return ""
}

// Username returns authenticated username in case authentication process has been completed.
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

// UsesChannelBinding returns whether or not scram authenticator requires channel binding bytes.
func (s *Scram) UsesChannelBinding() bool {
	return s.usesCb
}

// ProcessElement process an incoming authenticator element.
func (s *Scram) ProcessElement(ctx context.Context, elem stravaganza.Element) (stravaganza.Element, *SASLError) {
	switch elem.Name() {
	case "auth":
		if s.state == startScramState {
			return s.handleStart(ctx, elem)
		}
	case "response":
		if s.state == challengedScramState {
			return s.handleChallenged(elem)
		}
	}
	return nil, newSASLError(NotAuthorized, nil)
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

func (s *Scram) handleStart(ctx context.Context, elem stravaganza.Element) (stravaganza.Element, *SASLError) {
	p, saslErr := s.getElementPayload(elem)
	if saslErr != nil {
		return nil, saslErr
	}
	if saslErr := s.parseParameters(p); saslErr != nil {
		return nil, saslErr
	}
	username := s.params.getParameter("n")
	cNonce := s.params.getParameter("r")

	if len(username) == 0 || len(cNonce) == 0 {
		return nil, newSASLError(MalformedRequest, nil)
	}
	user, err := s.rep.FetchUser(ctx, username)
	if err != nil {
		return nil, newSASLError(TemporaryAuthFailure, err)
	}
	if user == nil {
		return nil, newSASLError(NotAuthorized, nil)
	}
	s.user = user

	saltBytes, err := base64.RawURLEncoding.DecodeString(user.Scram.Salt)
	if err != nil {
		return nil, newSASLError(TemporaryAuthFailure, err)
	}
	buf := bytes.NewBuffer(saltBytes)
	buf.WriteString(s.peppers.GetKey(user.Scram.PepperID))

	pepperedSaltB64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	s.srvNonce = cNonce + "-" + uuid.New().String()
	s.firstMessage = fmt.Sprintf("r=%s,s=%s,i=%d", s.srvNonce, pepperedSaltB64, user.Scram.IterationCount)

	s.state = challengedScramState

	return stravaganza.NewBuilder("challenge").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithText(base64.StdEncoding.EncodeToString([]byte(s.firstMessage))).
		Build(), nil
}

func (s *Scram) handleChallenged(elem stravaganza.Element) (stravaganza.Element, *SASLError) {
	p, saslErr := s.getElementPayload(elem)
	if saslErr != nil {
		return nil, saslErr
	}
	c := s.getCBindInputString()
	initialMessage := s.params.String()
	clientFinalMessageBare := fmt.Sprintf("c=%s,r=%s", c, s.srvNonce)

	scramPassword, err := s.getScramPassword()
	if err != nil {
		return nil, newSASLError(TemporaryAuthFailure, err)
	}
	clientKey := s.hmac([]byte("Client Key"), scramPassword)
	storedKey := s.hash(clientKey)
	authMessage := initialMessage + "," + s.firstMessage + "," + clientFinalMessageBare
	clientSignature := s.hmac([]byte(authMessage), storedKey)

	clientProof := make([]byte, len(clientKey))
	for i := 0; i < len(clientKey); i++ {
		clientProof[i] = clientKey[i] ^ clientSignature[i]
	}
	serverKey := s.hmac([]byte("Server Key"), scramPassword)
	serverSignature := s.hmac([]byte(authMessage), serverKey)

	clientFinalMessage := clientFinalMessageBare + ",p=" + base64.StdEncoding.EncodeToString(clientProof)
	if clientFinalMessage != p {
		return nil, newSASLError(NotAuthorized, err)
	}
	v := "v=" + base64.StdEncoding.EncodeToString(serverSignature)

	s.authenticated = true

	return stravaganza.NewBuilder("success").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithText(base64.StdEncoding.EncodeToString([]byte(v))).
		Build(), nil
}

func (s *Scram) getElementPayload(elem stravaganza.Element) (string, *SASLError) {
	if len(elem.Text()) == 0 {
		return "", newSASLError(IncorrectEncoding, nil)
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return "", newSASLError(IncorrectEncoding, err)
	}
	return string(b), nil
}

func (s *Scram) parseParameters(str string) *SASLError {
	p := &scramParameters{}

	sp := strings.Split(str, ",")
	if len(sp) < 2 {
		return newSASLError(IncorrectEncoding, nil)
	}
	gs2BindFlag := sp[0]

	// https://tools.ietf.org/html/rfc5801#section-5
	switch gs2BindFlag {
	case "p":
		// Channel binding is supported and required.
		if !s.usesCb {
			return newSASLError(NotAuthorized, nil)
		}
	case "n", "y":
		// Channel binding is not supported, or is supported but is not required.
		break
	default:
		if !strings.HasPrefix(gs2BindFlag, "p=") {
			return newSASLError(MalformedRequest, nil)
		}
		if !s.usesCb {
			return newSASLError(NotAuthorized, nil)
		}
		p.cbMechanism = gs2BindFlag[2:]
	}
	authzID := sp[1]
	p.gs2Header = gs2BindFlag + "," + authzID + ","

	if len(authzID) > 0 {
		key, val := stringsutil.SplitKeyAndValue(authzID, '=')
		if len(key) == 0 || key != "a" {
			return newSASLError(MalformedRequest, nil)
		}
		p.authzID = val
	}
	for i := 2; i < len(sp); i++ {
		key, val := stringsutil.SplitKeyAndValue(sp[i], '=')
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

func (s *Scram) getScramPassword() ([]byte, error) {
	var scramPass string
	switch s.tp {
	case ScramSHA1:
		scramPass = s.user.Scram.SHA1
	case ScramSHA256:
		scramPass = s.user.Scram.SHA256
	case ScramSHA512:
		scramPass = s.user.Scram.SHA512
	case ScramSHA3512:
		scramPass = s.user.Scram.SHA3512
	}
	return base64.RawURLEncoding.DecodeString(scramPass)
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
