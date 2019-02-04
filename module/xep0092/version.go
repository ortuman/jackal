/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0092

import (
	"os/exec"
	"strings"

	"github.com/ortuman/jackal/router"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

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

// Version represents a version module.
type Version struct {
	cfg        *Config
	router     *router.Router
	actorCh    chan func()
	shutdownCh chan chan error
}

// New returns a version IQ handler module.
func New(config *Config, disco *xep0030.DiscoInfo, router *router.Router) *Version {
	v := &Version{
		cfg:        config,
		router:     router,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan error),
	}
	go v.loop()
	if disco != nil {
		disco.RegisterServerFeature(versionNamespace)
	}
	return v
}

// MatchesIQ returns whether or not an IQ should be
// processed by the version module.
func (x *Version) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", versionNamespace) != nil && iq.ToJID().IsServer()
}

// ProcessIQ processes a version IQ taking according actions
// over the associated stream.
func (x *Version) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() {
		x.processIQ(iq)
	}
}

// Shutdown shuts down version module.
func (x *Version) Shutdown() error {
	c := make(chan error)
	x.shutdownCh <- c
	return <-c
}

// runs on it's own goroutine
func (x *Version) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- nil
			return
		}
	}
}

func (x *Version) processIQ(iq *xmpp.IQ) {
	q := iq.Elements().ChildNamespace("query", versionNamespace)
	if q == nil || q.Elements().Count() != 0 {
		_ = x.router.Route(iq.BadRequestError())
		return
	}
	x.sendSoftwareVersion(iq)
}

func (x *Version) sendSoftwareVersion(iq *xmpp.IQ) {
	userJID := iq.FromJID()
	username := userJID.Node()
	resource := userJID.Resource()
	log.Infof("retrieving software version: %v (%s/%s)", version.ApplicationVersion, username, resource)

	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", versionNamespace)

	name := xmpp.NewElementName("name")
	name.SetText("jackal")
	query.AppendElement(name)

	ver := xmpp.NewElementName("version")
	ver.SetText(version.ApplicationVersion.String())
	query.AppendElement(ver)

	if x.cfg.ShowOS {
		os := xmpp.NewElementName("os")
		os.SetText(osString)
		query.AppendElement(os)
	}
	result.AppendElement(query)
	_ = x.router.Route(result)
}
