/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"encoding/base64"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
)

type PlainAuthenticator struct {
	strm          *stream.Stream
	username      string
	authenticated bool
}

func NewPlainAuthenticator(strm *Stream) *PlainAuthenticator {
	return &PlainAuthenticator{strm: strm}
}

func (p *PlainAuthenticator) Mechanism() string {
	return "PLAIN"
}

func (p *PlainAuthenticator) Username() string {
	return p.username
}

func (p *PlainAuthenticator) Authenticated() bool {
	return p.authenticated
}

func (p *PlainAuthenticator) UsesChannelBinding() bool {
	return false
}

func (p *PlainAuthenticator) ProcessElement(elem *xml.Element) error {
	if p.authenticated {
		return nil
	}
	b64Payload := elem.Text()
	if len(b64Payload) == 0 {
		return InvalidFormatErr
	}
	b, err := base64.StdEncoding.DecodeString(b64Payload)
	if err != nil {
		return InvalidFormatErr
	}
	s := bytes.Split(b, []byte{0})
	if len(s) != 2 {
		return InvalidFormatErr
	}
	username := string(s[0])
	// password := string(s[1])

	// TODO: Validate user and password.

	p.username = username
	p.authenticated = true

	p.strm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}

func (p *PlainAuthenticator) Reset() {
	p.username = ""
	p.authenticated = false
}
