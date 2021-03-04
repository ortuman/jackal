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
	"crypto/sha1"
	"testing"

	"github.com/ortuman/jackal/module/xep0004"

	discomodel "github.com/ortuman/jackal/model/disco"
	"github.com/stretchr/testify/require"
)

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
