// Copyright 2021 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0115

import (
	"context"
	"crypto/sha1"
	"testing"
	"time"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/event"
	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
	discomodel "github.com/ortuman/jackal/pkg/model/disco"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestCapabilities_RequestDiscoInfo(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.CapabilitiesExistFunc = func(ctx context.Context, node string, ver string) (bool, error) {
		return false, nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	sn := sonar.New()
	c := &Capabilities{
		rep:    repMock,
		router: routerMock,
		sn:     sn,
		reqs:   make(map[string]capsInfo),
		clrTms: make(map[string]*time.Timer),
	}
	// when
	_ = c.Start(context.Background())
	defer func() { _ = c.Stop(context.Background()) }()

	jd0, _ := jid.NewWithString("noelia@jackal.im/yard", true)
	jd1, _ := jid.NewWithString("ortuman@jackal.im", true)

	cElem := stravaganza.NewBuilder("c").
		WithAttribute(stravaganza.Namespace, capabilitiesFeature).
		WithAttribute("hash", "sha-1").
		WithAttribute("node", "http://dino.im").
		WithAttribute("ver", "q07IKJEyjvHSyhy//CH0CxmKi8w=").
		Build()

	pr := xmpputil.MakePresence(jd0, jd1, stravaganza.AvailableType, []stravaganza.Element{cElem})
	_ = sn.Post(context.Background(),
		sonar.NewEventBuilder(event.C2SStreamPresenceReceived).
			WithInfo(&event.C2SStreamEventInfo{
				Element: pr,
			}).
			Build(),
	)

	// then
	require.Len(t, respStanzas, 1)

	elem := respStanzas[0]
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, stravaganza.GetType, elem.Attribute(stravaganza.Type))

	q := elem.ChildNamespace("query", discoInfoNamespace)
	require.Equal(t, "http://dino.im#q07IKJEyjvHSyhy//CH0CxmKi8w=", q.Attribute("node"))
}

func TestCapabilities_ProcessDiscoInfo(t *testing.T) {
	// given
	repMock := &repositoryMock{}

	var recvCaps *capsmodel.Capabilities
	repMock.UpsertCapabilitiesFunc = func(ctx context.Context, caps *capsmodel.Capabilities) error {
		recvCaps = caps
		return nil
	}
	routerMock := &routerMock{}

	sn := sonar.New()
	c := &Capabilities{
		rep:    repMock,
		router: routerMock,
		sn:     sn,
		reqs:   make(map[string]capsInfo),
		clrTms: make(map[string]*time.Timer),
	}
	c.reqs["id1234"] = capsInfo{
		node: "http://dino.im",
		ver:  "14j4+I88rSOWIY4WwJiIYgYqXrI=",
		hash: "sha-1",
	}

	discoIQ, _ := stravaganza.NewBuilder("iq").
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.Type, stravaganza.ResultType).
		WithAttribute(stravaganza.From, "noelia@jackal.im/yard").
		WithAttribute(stravaganza.To, "jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				WithChild(
					stravaganza.NewBuilder("feature").
						WithAttribute("var", "http://jabber.org/protocol/disco#info").
						Build(),
				).
				WithChild(
					stravaganza.NewBuilder("feature").
						WithAttribute("var", "http://jabber.org/protocol/disco#items").
						Build(),
				).
				Build(),
		).
		BuildIQ()

	// when
	_ = c.Start(context.Background())
	defer func() { _ = c.Stop(context.Background()) }()

	_ = sn.Post(context.Background(),
		sonar.NewEventBuilder(event.C2SStreamIQReceived).
			WithInfo(&event.C2SStreamEventInfo{
				Element: discoIQ,
			}).
			Build(),
	)

	// then
	require.NotNil(t, recvCaps)

	require.Equal(t, "http://dino.im", recvCaps.Node)
	require.Equal(t, "14j4+I88rSOWIY4WwJiIYgYqXrI=", recvCaps.Ver)

	require.Len(t, recvCaps.Features, 2)
}

func TestCapabilities_ComputeSimpleVerificationString(t *testing.T) {
	// given
	identities := []discomodel.Identity{
		{Category: "client", Type: "pc", Name: "Exodus 0.9.1"},
	}
	features := []discomodel.Feature{
		"http://jabber.org/protocol/disco#info",
		"http://jabber.org/protocol/disco#items",
		"http://jabber.org/protocol/muc",
		"http://jabber.org/protocol/caps",
	}
	// when
	ver := computeVer(identities, features, nil, sha1.New)

	// then
	require.Equal(t, "QgayPKawpkPSDYmwT/WM94uAlu0=", ver)
}

func TestCapabilities_ComputeComplexVerificationString(t *testing.T) {
	// given
	identities := []discomodel.Identity{
		{Category: "client", Type: "pc", Name: "Ψ 0.11", Lang: "el"},
		{Category: "client", Type: "pc", Name: "Psi 0.11", Lang: "en"},
	}
	features := []discomodel.Feature{
		"http://jabber.org/protocol/disco#info",
		"http://jabber.org/protocol/disco#items",
		"http://jabber.org/protocol/muc",
		"http://jabber.org/protocol/caps",
	}
	forms := []xep0004.DataForm{
		{
			Type: xep0004.Result,
			Fields: xep0004.Fields{
				{
					Var:    xep0004.FormType,
					Type:   xep0004.Hidden,
					Values: []string{"urn:xmpp:dataforms:softwareinfo"},
				},
				{
					Var:    "ip_version",
					Type:   xep0004.TextMulti,
					Values: []string{"ipv4", "ipv6"},
				},
				{
					Var:    "os",
					Values: []string{"Mac"},
				},
				{
					Var:    "os_version",
					Values: []string{"10.5.1"},
				},
				{
					Var:    "software",
					Values: []string{"Psi"},
				},
				{
					Var:    "software_version",
					Values: []string{"0.11"},
				},
			},
		},
	}
	// when
	ver := computeVer(identities, features, forms, sha1.New)

	// then
	require.Equal(t, "q07IKJEyjvHSyhy//CH0CxmKi8w=", ver)
}
