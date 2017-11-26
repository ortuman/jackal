/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/xml"

type scramAuthenticator struct {
}

func (s *scramAuthenticator) Mechanism() string {
	return "SCRAM-SHA-1"
}

func (s *scramAuthenticator) Username() string {
	return ""
}

func (s *scramAuthenticator) Authenticated() bool {
	return false
}

func (s *scramAuthenticator) UsesChannelBinding() bool {
	return false
}

func (s *scramAuthenticator) ProcessElement(elem *xml.Element) error {
	return nil
}

func (s *scramAuthenticator) Reset() {
}
