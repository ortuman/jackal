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

package xep0191

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/jid"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/stretchr/testify/require"
)

func TestBlockList_GetPresenceTargets(t *testing.T) {
	// given
	rep := &repositoryMock{}
	rep.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]rostermodel.Item, error) {
		return []rostermodel.Item{
			{Username: "ortuman", JID: "juliet@jabber.org", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "hamlet@jabber.org", Subscription: rostermodel.To},
			{Username: "ortuman", JID: "hamlet@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "macbeth@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@404.city", Subscription: rostermodel.To},
			{Username: "ortuman", JID: "witch@jackal.im", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@jabber.net", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@jabber.org", Subscription: rostermodel.To},
		}, nil
	}
	// when
	bl := &BlockList{
		rep: rep,
	}
	jd0, _ := jid.NewWithString("404.city/yard", true)
	jd1, _ := jid.NewWithString("jabber.org", true)
	jd2, _ := jid.NewWithString("witch@jackal.im", true)
	jd3, _ := jid.NewWithString("witch@jabber.net/chamber", true)

	pss, _ := bl.getPresenceTargets(context.Background(), []jid.JID{*jd0, *jd1, *jd2, *jd3}, "ortuman")

	// then
	require.Len(t, pss, 5)

	require.Equal(t, pss[0].String(), "hamlet@404.city/yard")
	require.Equal(t, pss[1].String(), "macbeth@404.city/yard")
	require.Equal(t, pss[2].String(), "juliet@jabber.org")
	require.Equal(t, pss[3].String(), "witch@jackal.im")
	require.Equal(t, pss[4].String(), "witch@jabber.net/chamber")
}
