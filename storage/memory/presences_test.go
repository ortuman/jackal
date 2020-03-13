package memorystorage

import (
	"context"
	"testing"

	capsmodel "github.com/ortuman/jackal/model/capabilities"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_FetchPresencesMatchingJID(t *testing.T) {
	const allocID = "1234"

	j1, _ := jid.NewWithString("noelia@jackal.im/garden", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	j3, _ := jid.NewWithString("noelia@jackal.im/yard", true)
	j4, _ := jid.NewWithString("boss@jabber.org/balcony", true)

	p1 := xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.AvailableType)
	p2 := xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.AvailableType)
	p3 := xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.AvailableType)
	p4 := xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.AvailableType)

	s := NewPresences()
	ok, err := s.UpsertPresence(context.Background(), p1, j1, allocID)
	require.True(t, ok)
	require.Nil(t, err)

	ok, err = s.UpsertPresence(context.Background(), p2, j2, allocID)
	require.True(t, ok)
	require.Nil(t, err)

	ok, err = s.UpsertPresence(context.Background(), p3, j3, allocID)
	require.True(t, ok)
	require.Nil(t, err)

	ok, err = s.UpsertPresence(context.Background(), p4, j4, allocID)
	require.True(t, ok)
	require.Nil(t, err)

	// updating presence
	ok, err = s.UpsertPresence(context.Background(), p1, j1, allocID)
	require.False(t, ok)
	require.Nil(t, err)

	mJID, _ := jid.NewWithString("jackal.im", true)
	presences, _ := s.FetchPresencesMatchingJID(context.Background(), mJID)
	require.Len(t, presences, 3)

	mJID, _ = jid.NewWithString("jackal.im/yard", true)
	presences, _ = s.FetchPresencesMatchingJID(context.Background(), mJID)
	require.Len(t, presences, 2)

	mJID, _ = jid.NewWithString("jabber.org", true)
	presences, _ = s.FetchPresencesMatchingJID(context.Background(), mJID)
	require.Len(t, presences, 1)

	_ = s.DeletePresence(context.Background(), j2)
	mJID, _ = jid.NewWithString("jackal.im/yard", true)
	presences, _ = s.FetchPresencesMatchingJID(context.Background(), mJID)
	require.Len(t, presences, 1)

	_ = s.ClearPresences(context.Background())
	mJID, _ = jid.NewWithString("jackal.im", true)
	presences, _ = s.FetchPresencesMatchingJID(context.Background(), mJID)
	require.Len(t, presences, 0)
}

func TestMemoryStorage_InsertCapabilities(t *testing.T) {
	caps := capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := NewPresences()
	EnableMockedError()
	err := s.UpsertCapabilities(context.Background(), &caps)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()
	err = s.UpsertCapabilities(context.Background(), &caps)
	require.Nil(t, err)
}

func TestMemoryStorage_FetchCapabilities(t *testing.T) {
	caps := capsmodel.Capabilities{Node: "n1", Ver: "1234A", Features: []string{"ns"}}
	s := NewPresences()
	_ = s.UpsertCapabilities(context.Background(), &caps)

	EnableMockedError()
	_, err := s.FetchCapabilities(context.Background(), "n1", "1234A")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	cs, _ := s.FetchCapabilities(context.Background(), "n1", "1234B")
	require.Nil(t, cs)

	cs, _ = s.FetchCapabilities(context.Background(), "n1", "1234A")
	require.NotNil(t, cs)
}
