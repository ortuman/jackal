/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"bytes"
	"encoding/base64"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

// Plain represents a PLAIN authenticator.
type Plain struct {
	stm           stream.C2S
	username      string
	authenticated bool
}

// NewPlain returns a new plain authenticator instance.
func NewPlain(stm stream.C2S) *Plain {
	return &Plain{stm: stm}
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
func (p *Plain) ProcessElement(elem xmpp.XElement) error {
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
	password := string(s[2])

	// validate user and password
	user, err := storage.FetchUser(username)
	if err != nil {
		return err
	}
	if user == nil || user.Password != password {
		return ErrSASLNotAuthorized
	}
	p.username = username
	p.authenticated = true

	p.stm.SendElement(xmpp.NewElementNamespace("success", saslNamespace))
	return nil
}

// Reset resets plain authenticator internal state.
func (p *Plain) Reset() {
	p.username = ""
	p.authenticated = false
}
