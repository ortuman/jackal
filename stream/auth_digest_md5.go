/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/xml"

type digestMD5State int

const (
	digestMD5StateStart digestMD5State = iota
	digestMD5StateChallenged
	digestMD5StateAuthenticated
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
		case digestMD5StateStart:
			return d.handleStart(elem)
		}
	case "response":
		switch d.state {
		case digestMD5StateChallenged:
			return d.handleChallenged(elem)
		case digestMD5StateAuthenticated:
			return d.handleAuthenticated(elem)
		}
	}
	return errSASLNotAuthorized
}

func (d *digestMD5Authenticator) Reset() {
	d.state = digestMD5Start
	d.username = ""
	d.authenticated = false
}

func (d *digestMD5Authenticator) handleStart(elem *xml.Element) error {
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
