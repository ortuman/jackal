/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestOffline_ArchiveMessage(t *testing.T) {
	r := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("juliet", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm)

	x := New(&Config{QueueSize: 1}, nil, r)
	defer func() { _ = x.Shutdown() }()

	msgID := uuid.New()
	msg := xmpp.NewMessageType(msgID, "normal")
	msg.SetFromJID(j1)
	msg.SetToJID(j2)
	x.ArchiveMessage(context.Background(), msg)

	// wait for insertion...
	time.Sleep(time.Millisecond * 250)

	msgs, err := storage.FetchOfflineMessages(context.Background(), "juliet")
	require.Nil(t, err)
	require.Equal(t, 1, len(msgs))

	msg2 := xmpp.NewMessageType(msgID, "normal")
	msg2.SetFromJID(j1)
	msg2.SetToJID(j2)

	x.ArchiveMessage(context.Background(), msg)

	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, xmpp.ErrServiceUnavailable.Error(), elem.Error().Elements().All()[0].Name())

	// deliver offline messages...
	stm2 := stream.NewMockC2S("abcd", j2)
	r.Bind(context.Background(), stm2)

	x2 := New(&Config{QueueSize: 1}, nil, r)
	defer func() { _ = x.Shutdown() }()

	x2.DeliverOfflineMessages(context.Background(), stm2)

	elem = stm2.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, msgID, elem.ID())
}

func setupTest(domain string) *router.Router {
	storage.Unset()
	s2 := memorystorage.New2()
	storage.Set(s2)

	s := memorystorage.NewUser()
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
	}, s)
	return r
}
