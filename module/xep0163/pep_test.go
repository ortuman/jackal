package xep0163

import (
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0163_Matching(t *testing.T) {
	r, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(stm)

	p := New(nil, r)

	// test MatchesIQ
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j)
	iq.AppendElement(xmpp.NewElementNamespace("pubsub", pepNamespace))
	require.True(t, p.MatchesIQ(iq))
}

func TestXEP163_CreateNode(t *testing.T) {
	r, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(stm)

	p := New(nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j)

	pubSub := xmpp.NewElementNamespace("pubsub", pepNamespace)
	create := xmpp.NewElementName("create")
	create.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(create)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())
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
