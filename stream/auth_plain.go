/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"encoding/base64"

	"github.com/ortuman/jackal/xml"
)

type plainAuthenticator struct {
	strm          *Stream
	username      string
	authenticated bool
}

func newPlainAuthenticator(strm *Stream) authenticator {
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

func (p *plainAuthenticator) ProcessElement(elem *xml.Element) error {
	if p.authenticated {
		return nil
	}
	b64Payload := elem.Text()
	if len(b64Payload) == 0 {
		return errInvalidFormat
	}
	b, err := base64.StdEncoding.DecodeString(b64Payload)
	if err != nil {
		return errInvalidFormat
	}
	s := bytes.Split(b, []byte{0})
	if len(s) != 2 {
		return errInvalidFormat
	}
	username := string(s[0])
	// password := string(s[1])

	// TODO: Validate user and password.

	p.username = username
	p.authenticated = true

	p.strm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}

func (p *plainAuthenticator) Reset() {
	p.username = ""
	p.authenticated = false
}
