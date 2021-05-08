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

package offline

import (
	"bytes"
	"context"
	"testing"

	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/cluster/locker"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/module"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestOffline_ArchiveOfflineMessage(t *testing.T) {
	// given
	lockMock := &lockMock{}
	lockMock.ReleaseFunc = func(ctx context.Context) error {
		return nil
	}
	lockerMock := &lockerMock{}
	lockerMock.AcquireLockFunc = func(ctx context.Context, lockID string) (locker.Lock, error) {
		return lockMock, nil
	}
	hostsMock := &hostsMock{}
	hostsMock.IsLocalHostFunc = func(h string) bool { return h == "jackal.im" }

	resManagerMock := &resourceManagerMock{}
	resManagerMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.Resource, error) {
		return nil, nil
	}
	repMock := &repositoryMock{}
	repMock.UserExistsFunc = func(ctx context.Context, username string) (bool, error) {
		return true, nil
	}
	repMock.CountOfflineMessagesFunc = func(ctx context.Context, username string) (int, error) {
		return 0, nil
	}
	repMock.InsertOfflineMessageFunc = func(ctx context.Context, message *stravaganza.Message, username string) error {
		return nil
	}
	sn := sonar.New()

	m := &Offline{
		cfg:    Config{QueueSize: 100},
		rep:    repMock,
		hosts:  hostsMock,
		resMng: resManagerMock,
		locker: lockerMock,
		sn:     sn,
	}
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = m.Start(context.Background())

	stanza, err := m.InterceptStanza(context.Background(), msg, 0)

	// then
	require.Equal(t, module.ErrInterceptionInterrupted, err)
	require.Nil(t, stanza)

	require.Len(t, repMock.CountOfflineMessagesCalls(), 1)
	require.Len(t, repMock.InsertOfflineMessageCalls(), 1)
}

func TestOffline_ArchiveOfflineMessageQueueFull(t *testing.T) {
	// given
	routerMock := &routerMock{}

	output := bytes.NewBuffer(nil)
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		_ = stanza.ToXML(output, true)
		return nil, nil
	}
	hostsMock := &hostsMock{}
	hostsMock.IsLocalHostFunc = func(h string) bool { return h == "jackal.im" }

	resManagerMock := &resourceManagerMock{}
	resManagerMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.Resource, error) {
		return nil, nil
	}

	lockMock := &lockMock{}
	lockMock.ReleaseFunc = func(ctx context.Context) error {
		return nil
	}
	lockerMock := &lockerMock{}
	lockerMock.AcquireLockFunc = func(ctx context.Context, lockID string) (locker.Lock, error) {
		return lockMock, nil
	}
	repMock := &repositoryMock{}
	repMock.UserExistsFunc = func(ctx context.Context, username string) (bool, error) {
		return true, nil
	}
	repMock.CountOfflineMessagesFunc = func(ctx context.Context, username string) (int, error) {
		return 100, nil
	}
	repMock.InsertOfflineMessageFunc = func(ctx context.Context, message *stravaganza.Message, username string) error {
		return nil
	}
	sn := sonar.New()

	m := &Offline{
		cfg:    Config{QueueSize: 100},
		router: routerMock,
		hosts:  hostsMock,
		resMng: resManagerMock,
		rep:    repMock,
		locker: lockerMock,
		sn:     sn,
	}
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = m.Start(context.Background())

	stanza, err := m.InterceptStanza(context.Background(), msg, 0)

	// then
	require.Equal(t, module.ErrInterceptionInterrupted, err)
	require.Nil(t, stanza)

	require.Len(t, repMock.CountOfflineMessagesCalls(), 1)
	require.Len(t, repMock.InsertOfflineMessageCalls(), 0)

	require.Equal(t, `<message from="ortuman@jackal.im/balcony" to="noelia@jackal.im/yard" type="error"><body>I&#39;ll give thee a wind.</body><error code="503" type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></error></message>`, output.String())
}

func TestOffline_DeliverOfflineMessages(t *testing.T) {
	// given
	hostsMock := &hostsMock{}
	hostsMock.IsLocalHostFunc = func(h string) bool { return h == "jackal.im" }

	resManagerMock := &resourceManagerMock{}
	resManagerMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.Resource, error) {
		return nil, nil
	}
	lockMock := &lockMock{}
	lockMock.ReleaseFunc = func(ctx context.Context) error {
		return nil
	}
	lockerMock := &lockerMock{}
	lockerMock.AcquireLockFunc = func(ctx context.Context, lockID string) (locker.Lock, error) {
		return lockMock, nil
	}
	repMock := &repositoryMock{}
	repMock.CountOfflineMessagesFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	repMock.FetchOfflineMessagesFunc = func(ctx context.Context, username string) ([]*stravaganza.Message, error) {
		b := stravaganza.NewMessageBuilder()
		b.WithAttribute("from", "noelia@jackal.im/yard")
		b.WithAttribute("to", "ortuman@jackal.im/balcony")
		b.WithChild(
			stravaganza.NewBuilder("body").
				WithText("I'll give thee a wind.").
				Build(),
		)
		msg, _ := b.BuildMessage()

		return []*stravaganza.Message{msg}, nil
	}
	repMock.DeleteOfflineMessagesFunc = func(ctx context.Context, username string) error {
		return nil
	}
	streamMock := &streamMock{}
	streamMock.UsernameFunc = func() string { return "ortuman" }
	streamMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		return nil
	}
	streamMock.InfoFunc = func() c2smodel.Info {
		return c2smodel.InfoFromMap(make(map[string]string))
	}
	output := bytes.NewBuffer(nil)
	streamMock.SendElementFunc = func(elem stravaganza.Element) <-chan error {
		_ = elem.ToXML(output, true)
		return nil
	}

	sn := sonar.New()
	m := &Offline{
		cfg:    Config{QueueSize: 100},
		hosts:  hostsMock,
		resMng: resManagerMock,
		rep:    repMock,
		locker: lockerMock,
		sn:     sn,
	}

	// when
	_ = m.Start(context.Background())

	fromJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	toJID, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.AvailableType, nil)

	_ = sn.Post(context.Background(),
		sonar.NewEventBuilder(event.C2SStreamPresenceReceived).
			WithInfo(&event.C2SStreamEventInfo{
				Element: pr,
			}).
			WithSender(streamMock).
			Build(),
	)

	// then
	require.Len(t, repMock.FetchOfflineMessagesCalls(), 1)
	require.Len(t, repMock.DeleteOfflineMessagesCalls(), 1)

	require.Equal(t, `<message from="noelia@jackal.im/yard" to="ortuman@jackal.im/balcony"><body>I&#39;ll give thee a wind.</body></message>`, output.String())
}
