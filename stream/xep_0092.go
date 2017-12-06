/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
)

const versionNamespace = "jabber:iq:version"

type xepVersion struct {
	cfg  *config.ModVersion
	strm *Stream
}

func newXepVersion(config *config.ModVersion, strm *Stream) *xepVersion {
	x := &xepVersion{cfg: config, strm: strm}
	return x
}

func (x *xepVersion) MatchesIQ(iq *xml.IQ) bool {
	return false
}

func (x *xepVersion) ProcessIQ(iq *xml.IQ) {
}
