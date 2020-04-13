/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"

	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

// Plain represents a PLAIN authenticator.
type Plain struct {
	stm           stream.C2S
	userRep       repository.User
	username      string
	authenticated bool
}

// NewPlain returns a new plain authenticator instance.
func NewPlain(stm stream.C2S, userRep repository.User) *Plain {
	return &Plain{stm: stm, userRep: userRep}
}

// Mechanism returns authenticator mechanism name.
func (p *Plain) Mechanism() string {
	return "PLAIN"
}

// Username returns authenticated username in case
// authentication process has been completed.
func (p *Plain) Username() string {
	return p.username
}

// Authenticated returns whether or not user has been authenticated.
func (p *Plain) Authenticated() bool {
	return p.authenticated
}

// UsesChannelBinding returns whether or not plain authenticator
// requires channel binding bytes.
func (p *Plain) UsesChannelBinding() bool {
	return false
}

// ProcessElement process an incoming authenticator element.
func (p *Plain) ProcessElement(ctx context.Context, elem xmpp.XElement) error {
	if p.authenticated {
		return nil
	}
	if len(elem.Text()) == 0 {
		return ErrSASLMalformedRequest
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return ErrSASLIncorrectEncoding
	}
	s := bytes.Split(b, []byte{0})
	if len(s) != 3 {
		return ErrSASLIncorrectEncoding
	}
	username := string(s[1])
	password := s[2]

	// validate user and password
	user, err := p.userRep.FetchUser(ctx, username)
	switch {
	case err != nil:
		return err
	case user == nil:
		return ErrSASLNotAuthorized
	}
	expectedPassword := SaltedPassword(password, user.Salt, user.IterationCount, sha256.New)
	if subtle.ConstantTimeCompare(user.PasswordScramSHA256, expectedPassword) != 1 {
		return ErrSASLNotAuthorized
	}
	p.username = username
	p.authenticated = true

	p.stm.SendElement(ctx, xmpp.NewElementNamespace("success", saslNamespace))
	return nil
}

// Reset resets plain authenticator internal state.
func (p *Plain) Reset() {
	p.username = ""
	p.authenticated = false
}
