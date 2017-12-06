/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"os/exec"
	"strings"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xml"
)

const versionNamespace = "jabber:iq:version"

var osString string

func init() {
	out, _ := exec.Command("uname", "-rs").Output()
	osString = strings.TrimSpace(string(out))
}

type xepVersion struct {
	cfg  *config.ModVersion
	strm *Stream
}

func newXepVersion(config *config.ModVersion, strm *Stream) *xepVersion {
	x := &xepVersion{cfg: config, strm: strm}
	return x
}

func (x *xepVersion) MatchesIQ(iq *xml.IQ) bool {
	return iq.IsGet() && iq.FindElementNamespace("query", versionNamespace) != nil && iq.ToJID().IsServer()
}

func (x *xepVersion) ProcessIQ(iq *xml.IQ) {
	q := iq.FindElementNamespace("query", versionNamespace)
	if q.ElementsCount() != 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	x.sendSoftwareVersion(iq)
}

func (x *xepVersion) sendSoftwareVersion(iq *xml.IQ) {
	log.Infof("retrieving software version: %v (username: %s)", version.ApplicationVersion, x.strm.Username())

	result := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", versionNamespace)

	name := xml.NewMutableElementName("name")
	name.SetText("jackal")
	query.AppendElement(name.Copy())

	ver := xml.NewMutableElementName("version")
	ver.SetText(version.ApplicationVersion.String())
	query.AppendElement(ver.Copy())

	if x.cfg.ShowOS {
		os := xml.NewMutableElementName("os")
		os.SetText(osString)
		query.AppendElement(os.Copy())
	}
	result.AppendElement(query.Copy())
	x.strm.SendElement(result)
}
