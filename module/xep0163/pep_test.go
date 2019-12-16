/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"crypto/tls"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/xep0004"
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

	p := New(nil, nil, r)

	// test MatchesIQ
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.AppendElement(xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace))

	require.True(t, p.MatchesIQ(iq))
}

func TestXEP163_CreateNode(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(stm)

	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	create := xmpp.NewElementName("create")
	create.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(create)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// read node
	n, _ := s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")
	require.NotNil(t, n)
	require.Equal(t, n.Options, defaultNodeOptions)
}

func TestXEP163_GetNodeConfiguration(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(stm)

	err := s.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	configureElem := xmpp.NewElementName("configure")
	configureElem.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(configureElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// get form element
	pubSubRes := elem.Elements().ChildNamespace("pubsub", pubSubOwnerNamespace)
	require.NotNil(t, pubSubRes)
	configElem := pubSubRes.Elements().Child("configure")
	require.NotNil(t, configElem)
	formEl := configElem.Elements().ChildNamespace("x", xep0004.FormNamespace)
	require.NotNil(t, formEl)

	configForm, err := xep0004.NewFormFromElement(formEl)
	require.Nil(t, err)
	require.Equal(t, xep0004.Form, configForm.Type)
}

func TestXEP163_SetNodeConfiguration(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(stm1)
	r.Bind(stm2)

	nodeOpts := defaultNodeOptions
	nodeOpts.NotifyConfig = true

	// create node and affiliations
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: nodeOpts,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		JID:          "ortuman@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	_, err = s.UpsertRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	configureElem := xmpp.NewElementName("configure")
	configureElem.SetAttribute("node", "princely_musings")

	// attach config update
	nodeOpts.Title = "a fancy new title"

	configForm := nodeOpts.ResultForm()
	configForm.Type = xep0004.Submit
	configureElem.AppendElement(configForm.Element())

	pubSub.AppendElement(configureElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)

	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "message", elem.Name()) // notification
	require.NotNil(t, elem.Elements().ChildNamespace("event", pubSubEventNamespace))

	elem2 := stm2.ReceiveElement()
	require.NotNil(t, elem2)
	require.Equal(t, "message", elem.Name()) // notification
	eventElem := elem2.Elements().ChildNamespace("event", pubSubEventNamespace)
	require.NotNil(t, eventElem)

	configElemResp := eventElem.Elements().Child("configuration")
	require.NotNil(t, configElemResp)
	require.Equal(t, "princely_musings", configElemResp.Attributes().Get("node"))

	// result IQ
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// check if configuration was applied
	n, _ := s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")
	require.NotNil(t, n)
	require.Equal(t, nodeOpts.Title, n.Options.Title)
}

func TestXEP163_DeleteNode(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(stm1)
	r.Bind(stm2)

	nodeOpts := defaultNodeOptions
	nodeOpts.NotifyDelete = true

	// create node and affiliations
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: nodeOpts,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		JID:          "ortuman@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	_, err = s.UpsertRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	deleteElem := xmpp.NewElementName("delete")
	deleteElem.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(deleteElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "message", elem.Name()) // notification
	require.NotNil(t, elem.Elements().ChildNamespace("event", pubSubEventNamespace))

	elem2 := stm2.ReceiveElement()
	require.NotNil(t, elem2)
	require.Equal(t, "message", elem.Name()) // notification
	eventElem := elem2.Elements().ChildNamespace("event", pubSubEventNamespace)
	require.NotNil(t, eventElem)

	deleteElemResp := eventElem.Elements().Child("delete")
	require.NotNil(t, deleteElemResp)
	require.Equal(t, "princely_musings", deleteElemResp.Attributes().Get("node"))

	// result IQ
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// read node
	n, _ := s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")
	require.Nil(t, n)
}

func TestXEP163_UpdateAffiliations(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(stm1)

	// create node
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	// create new affiliation
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	affElem := xmpp.NewElementName("affiliations")
	affElem.SetAttribute("node", "princely_musings")

	affiliation := xmpp.NewElementName("affiliation")
	affiliation.SetAttribute("jid", "noelia@jackal.im")
	affiliation.SetAttribute("affiliation", pubsubmodel.Owner)
	affElem.AppendElement(affiliation)
	pubSub.AppendElement(affElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	aff, _ := s.FetchPubSubNodeAffiliation("ortuman@jackal.im", "princely_musings", "noelia@jackal.im")
	require.NotNil(t, aff)
	require.Equal(t, "noelia@jackal.im", aff.JID)
	require.Equal(t, pubsubmodel.Owner, aff.Affiliation)

	// remove affiliation
	affiliation.SetAttribute("affiliation", pubsubmodel.None)

	p.ProcessIQ(iq)
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	aff, _ = s.FetchPubSubNodeAffiliation("ortuman@jackal.im", "princely_musings", "noelia@jackal.im")
	require.Nil(t, aff)
}

func TestXEP163_RetrieveAffiliations(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(stm1)

	// create node and affiliations
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	affElem := xmpp.NewElementName("affiliations")
	affElem.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(affElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())

	pubSubElem := elem.Elements().ChildNamespace("pubsub", pubSubOwnerNamespace)
	require.NotNil(t, pubSubElem)

	affiliationsElem := pubSubElem.Elements().Child("affiliations")
	require.NotNil(t, affiliationsElem)

	affiliations := affiliationsElem.Elements().Children("affiliation")
	require.Len(t, affiliations, 2)

	require.Equal(t, "ortuman@jackal.im", affiliations[0].Attributes().Get("jid"))
	require.Equal(t, pubsubmodel.Owner, affiliations[0].Attributes().Get("affiliation"))
	require.Equal(t, "noelia@jackal.im", affiliations[1].Attributes().Get("jid"))
	require.Equal(t, pubsubmodel.Owner, affiliations[1].Attributes().Get("affiliation"))
}

func TestXEP163_UpdateSubscriptions(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(stm1)

	// create node
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	// create new subscription
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	subElem := xmpp.NewElementName("subscriptions")
	subElem.SetAttribute("node", "princely_musings")

	sub := xmpp.NewElementName("subscription")
	sub.SetAttribute("jid", "noelia@jackal.im")
	sub.SetAttribute("subscription", pubsubmodel.Subscribed)
	subElem.AppendElement(sub)
	pubSub.AppendElement(subElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	subs, _ := s.FetchPubSubNodeSubscriptions("ortuman@jackal.im", "princely_musings")
	require.NotNil(t, subs)
	require.Len(t, subs, 1)
	require.Equal(t, "noelia@jackal.im", subs[0].JID)
	require.Equal(t, pubsubmodel.Subscribed, subs[0].Subscription)

	// remove subscription
	sub.SetAttribute("subscription", pubsubmodel.None)

	p.ProcessIQ(iq)
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	subs, _ = s.FetchPubSubNodeSubscriptions("ortuman@jackal.im", "princely_musings")
	require.Nil(t, subs)
}

func TestXEP163_RetrieveSubscriptions(t *testing.T) {
	r, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(stm1)

	// create node and affiliations
	err := storage.UpsertPubSubNode(&pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	require.Nil(t, err)

	err = s.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	err = s.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		SubID:        uuid.New(),
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	affElem := xmpp.NewElementName("subscriptions")
	affElem.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(affElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())

	pubSubElem := elem.Elements().ChildNamespace("pubsub", pubSubOwnerNamespace)
	require.NotNil(t, pubSubElem)

	subscriptionsElem := pubSubElem.Elements().Child("subscriptions")
	require.NotNil(t, subscriptionsElem)

	subscriptions := subscriptionsElem.Elements().Children("subscription")
	require.Len(t, subscriptions, 1)

	require.Equal(t, "noelia@jackal.im", subscriptions[0].Attributes().Get("jid"))
	require.Equal(t, pubsubmodel.Subscribed, subscriptions[0].Attributes().Get("subscription"))
}

func TestXEP163_Subscribe(t *testing.T) {
}

func TestXEP163_Unsubscribe(t *testing.T) {
}

func TestXEP163_RetrieveItems(t *testing.T) {
}

func TestXEP163_AutoSubscribe(t *testing.T) {
}

func TestXEP163_FilteredNotifications(t *testing.T) {
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
