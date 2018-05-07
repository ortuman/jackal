/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0092

import (
	"os/exec"
	"strings"

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

// Config represents XMPP Software Version module (XEP-0092) configuration.
type Config struct {
	ShowOS bool `yaml:"show_os"`
}

// XEPVersion represents a version server stream module.
type XEPVersion struct {
	cfg *Config
	stm c2s.Stream
}

// New returns a version IQ handler module.
func New(config *Config, stm c2s.Stream) *XEPVersion {
	x := &XEPVersion{
		cfg: config,
		stm: stm,
	}
	return x
}

// AssociatedNamespaces returns namespaces associated
// with version module.
func (x *XEPVersion) AssociatedNamespaces() []string {
	return []string{versionNamespace}
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
		x.stm.SendElement(iq.BadRequestError())
		return
	}
	x.sendSoftwareVersion(iq)
}

func (x *XEPVersion) sendSoftwareVersion(iq *xml.IQ) {
	username := x.stm.Username()
	resource := x.stm.Resource()
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
	x.stm.SendElement(result)
}
