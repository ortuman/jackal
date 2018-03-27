/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"os/exec"
	"strings"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xml"
)

const versionNamespace = "jabber:iq:version"

var osString string

func init() {
	out, _ := exec.Command("uname", "-rs").Output()
	osString = strings.TrimSpace(string(out))
}

// XEPVersion represents a version server stream module.
type XEPVersion struct {
	cfg  *config.ModVersion
	strm c2s.Stream
}

// NewXEPVersion returns a version IQ handler module.
func NewXEPVersion(config *config.ModVersion, strm c2s.Stream) *XEPVersion {
	x := &XEPVersion{
		cfg:  config,
		strm: strm,
	}
	return x
}

// AssociatedNamespaces returns namespaces associated
// with version module.
func (x *XEPVersion) AssociatedNamespaces() []string {
	return []string{versionNamespace}
}

// Done signals stream termination.
func (x *XEPVersion) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the version module.
func (x *XEPVersion) MatchesIQ(iq *xml.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", versionNamespace) != nil && iq.ToJID().IsServer()
}

// ProcessIQ processes a version IQ taking according actions
// over the associated stream.
func (x *XEPVersion) ProcessIQ(iq *xml.IQ) {
	q := iq.Elements().ChildNamespace("query", versionNamespace)
	if q.Elements().Count() != 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	x.sendSoftwareVersion(iq)
}

func (x *XEPVersion) sendSoftwareVersion(iq *xml.IQ) {
	username := x.strm.Username()
	resource := x.strm.Resource()
	log.Infof("retrieving software version: %v (%s/%s)", version.ApplicationVersion, username, resource)

	result := iq.ResultIQ()
	query := xml.NewElementNamespace("query", versionNamespace)

	name := xml.NewElementName("name")
	name.SetText("jackal")
	query.AppendElement(name)

	ver := xml.NewElementName("version")
	ver.SetText(version.ApplicationVersion.String())
	query.AppendElement(ver)

	if x.cfg.ShowOS {
		os := xml.NewElementName("os")
		os.SetText(osString)
		query.AppendElement(os)
	}
	result.AppendElement(query)
	x.strm.SendElement(result)
}
