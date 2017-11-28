/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"encoding/base64"
	"fmt"

	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
)

type digestMD5State int

const (
	startDigestMD5State digestMD5State = iota
	challengedDigestMD5State
	authenticatedDigestMD5State
)

type digestMD5Authenticator struct {
	strm          *Stream
	state         digestMD5State
	username      string
	authenticated bool
}

func newDigestMD5(strm *Stream) authenticator {
	return &digestMD5Authenticator{strm: strm}
}

func (d *digestMD5Authenticator) Mechanism() string {
	return "DIGEST-MD5"
}

func (d *digestMD5Authenticator) Username() string {
	return d.username
}

func (d *digestMD5Authenticator) Authenticated() bool {
	return d.authenticated
}

func (d *digestMD5Authenticator) UsesChannelBinding() bool {
	return false
}

func (d *digestMD5Authenticator) ProcessElement(elem *xml.Element) error {
	if d.Authenticated() {
		return nil
	}
	switch elem.Name() {
	case "auth":
		switch d.state {
		case startDigestMD5State:
			return d.handleStart(elem)
		}
	case "response":
		switch d.state {
		case challengedDigestMD5State:
			return d.handleChallenged(elem)
		case authenticatedDigestMD5State:
			return d.handleAuthenticated(elem)
		}
	}
	return errSASLNotAuthorized
}

func (d *digestMD5Authenticator) Reset() {
	d.state = startDigestMD5State
	d.username = ""
	d.authenticated = false
}

func (d *digestMD5Authenticator) handleStart(elem *xml.Element) error {
	domain := d.strm.Domain()
	nonce := base64.StdEncoding.EncodeToString(util.RandomBytes(32))
	cg := fmt.Sprintf(`realm="%s",nonce="%s",qop="auth",charset=utf-8,algorithm=md5-sess`, domain, nonce)

	respElem := xml.NewMutableElementNamespace("challenge", saslNamespace)
	respElem.SetText(base64.StdEncoding.EncodeToString([]byte(cg)))
	d.strm.SendElement(respElem.Copy())

	d.state = challengedDigestMD5State
	return nil
}

func (d *digestMD5Authenticator) handleChallenged(elem *xml.Element) error {
	return nil
}

func (d *digestMD5Authenticator) handleAuthenticated(elem *xml.Element) error {
	d.authenticated = true
	d.strm.SendElement(xml.NewElementNamespace("success", saslNamespace))
	return nil
}
