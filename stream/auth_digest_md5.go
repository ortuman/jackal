/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/xml"

type digestMD5Authenticator struct {
	strm          *Stream
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
	return nil
}

func (d *digestMD5Authenticator) Reset() {
}
