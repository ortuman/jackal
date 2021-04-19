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

package xep0012

import (
	"context"
	"testing"

	lastmodel "github.com/ortuman/jackal/model/last"

	"github.com/jackal-xmpp/stravaganza/v2"

	"github.com/jackal-xmpp/stravaganza/v2/jid"
	xmpputil "github.com/ortuman/jackal/util/xmpp"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/event"
	"github.com/stretchr/testify/require"
)

func TestLast_ProcessPresence(t *testing.T) {
	// given
	rep := &repositoryMock{}
	rep.UpsertLastFunc = func(ctx context.Context, last *lastmodel.Last) error {
		return nil
	}

	sn := sonar.New()
	bl := &Last{
		rep: rep,
		sn:  sn,
	}
	// when
	_ = bl.Start(context.Background())
	defer func() { _ = bl.Stop(context.Background()) }()

	jd0, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	_ = sn.Post(context.Background(), sonar.NewEventBuilder(event.C2SStreamPresenceReceived).
		WithInfo(&event.C2SStreamEventInfo{
			JID:    jd0,
			Stanza: xmpputil.MakePresence(jd0, jd0.ToBareJID(), stravaganza.UnavailableType, nil),
		}).
		Build(),
	)

	// then
	require.Len(t, rep.UpsertLastCalls(), 1)
}
