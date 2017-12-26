/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"os/exec"
	"strings"
	"time"

	"github.com/ortuman/jackal/concurrent"
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

type XEPVersion struct {
	queue concurrent.OperationQueue
	cfg   *config.ModVersion
	strm  Stream
}

func NewXEPVersion(config *config.ModVersion, strm Stream) *XEPVersion {
	x := &XEPVersion{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		cfg:  config,
		strm: strm,
	}
	return x
}

func (x *XEPVersion) AssociatedNamespaces() []string {
	return []string{versionNamespace}
}

func (x *XEPVersion) MatchesIQ(iq *xml.IQ) bool {
	return iq.IsGet() && iq.FindElementNamespace("query", versionNamespace) != nil && iq.ToJID().IsServer()
}

func (x *XEPVersion) ProcessIQ(iq *xml.IQ) {
	x.queue.Async(func() {
		q := iq.FindElementNamespace("query", versionNamespace)
		if q.ElementsCount() != 0 {
			x.strm.SendElement(iq.BadRequestError())
			return
		}
		x.sendSoftwareVersion(iq)
	})
}

func (x *XEPVersion) sendSoftwareVersion(iq *xml.IQ) {
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
