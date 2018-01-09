/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"bytes"
	"encoding/base64"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xml"
)

type plainAuthenticator struct {
	strm          *serverStream
	username      string
	authenticated bool
}

func newPlainAuthenticator(strm *serverStream) authenticator {
	return &plainAuthenticator{strm: strm}
}

func (p *plainAuthenticator) Mechanism() string {
	return "PLAIN"
}

func (p *plainAuthenticator) Username() string {
	return p.username
}

func (p *plainAuthenticator) Authenticated() bool {
	return p.authenticated
}

func (p *plainAuthenticator) UsesChannelBinding() bool {
	return false
}

func (p *plainAuthenticator) ProcessElement(elem xml.Element) error {
	if p.authenticated {
		return nil
	}
	if elem.TextLen() == 0 {
		return errSASLMalformedRequest
	}
	b, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return errSASLIncorrectEncoding
	}
	s := bytes.Split(b, []byte{0})
	if len(s) != 3 {
		return errSASLIncorrectEncoding
	}
	username := string(s[1])
	password := string(s[2])

	// validate user and password
	user, err := storage.Instance().FetchUser(username)
	if err != nil {
		return err
	}
	if user == nil || user.Password != password {
		return errSASLNotAuthorized
	}
	p.username = username
	p.authenticated = true

	p.strm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}

func (p *plainAuthenticator) Reset() {
	p.username = ""
	p.authenticated = false
}
