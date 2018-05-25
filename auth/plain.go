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
	"github.com/ortuman/jackal/xml"
)

type Plain struct {
	stm           stream.C2S
	username      string
	authenticated bool
}

func NewPlain(stm stream.C2S) *Plain {
	return &Plain{stm: stm}
}

func (p *Plain) Mechanism() string {
	return "PLAIN"
}

func (p *Plain) Username() string {
	return p.username
}

func (p *Plain) Authenticated() bool {
	return p.authenticated
}

func (p *Plain) UsesChannelBinding() bool {
	return false
}

func (p *Plain) ProcessElement(elem xml.XElement) error {
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
	user, err := storage.Instance().FetchUser(username)
	if err != nil {
		return err
	}
	if user == nil || user.Password != password {
		return ErrSASLNotAuthorized
	}
	p.username = username
	p.authenticated = true

	p.stm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}

func (p *Plain) Reset() {
	p.username = ""
	p.authenticated = false
}
