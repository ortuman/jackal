/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestOffline_ArchiveMessage(t *testing.T) {
	r, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("juliet", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(stm)

	x, shutdownCh := New(&Config{QueueSize: 1}, nil, r)
	defer close(shutdownCh)

	msgID := uuid.New()
	msg := xmpp.NewMessageType(msgID, "normal")
	msg.SetFromJID(j1)
	msg.SetToJID(j2)
	x.ArchiveMessage(msg)

	// wait for insertion...
	time.Sleep(time.Millisecond * 250)

	msgs, err := storage.FetchOfflineMessages("juliet")
	require.Nil(t, err)
	require.Equal(t, 1, len(msgs))

	msg2 := xmpp.NewMessageType(msgID, "normal")
	msg2.SetFromJID(j1)
	msg2.SetToJID(j2)

	x.ArchiveMessage(msg)

	elem := stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, xmpp.ErrServiceUnavailable.Error(), elem.Error().Elements().All()[0].Name())

	// deliver offline messages...
	stm2 := stream.NewMockC2S("abcd", j2)
	r.Bind(stm2)

	x2, _ := New(&Config{QueueSize: 1}, nil, r)
	x2.DeliverOfflineMessages(stm2)

	elem = stm2.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, msgID, elem.ID())
}

func setupTest(domain string) (*router.Router, *memstorage.Storage, func()) {
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
	})
	s := memstorage.New()
	storage.Set(s)
	return r, s, func() {
		storage.Unset()
	}
}
