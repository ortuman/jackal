/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/model"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/roster/presencehub"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0163_Matching(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	p := New(nil, nil, r)

	// test MatchesIQ
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.AppendElement(xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace))

	require.True(t, p.MatchesIQ(iq))
}

func TestXEP163_CreateNode(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

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

	p.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// read node
	n, _ := storage.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NotNil(t, n)
	require.Equal(t, n.Options, defaultNodeOptions)
}

func TestXEP163_GetNodeConfiguration(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

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

	p.ProcessIQ(context.Background(), iq)
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
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	nodeOpts := defaultNodeOptions
	nodeOpts.NotifyConfig = true

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: nodeOpts,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		JID:          "ortuman@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

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

	p.ProcessIQ(context.Background(), iq)

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
	n, _ := storage.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NotNil(t, n)
	require.Equal(t, nodeOpts.Title, n.Options.Title)
}

func TestXEP163_DeleteNode(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	nodeOpts := defaultNodeOptions
	nodeOpts.NotifyDelete = true

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: nodeOpts,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		JID:          "ortuman@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

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

	p.ProcessIQ(context.Background(), iq)
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
	n, _ := storage.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, n)
}

func TestXEP163_UpdateAffiliations(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	// create node
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

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

	p.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	aff, _ := storage.FetchNodeAffiliation(context.Background(), "ortuman@jackal.im", "princely_musings", "noelia@jackal.im")
	require.NotNil(t, aff)
	require.Equal(t, "noelia@jackal.im", aff.JID)
	require.Equal(t, pubsubmodel.Owner, aff.Affiliation)

	// remove affiliation
	affiliation.SetAttribute("affiliation", pubsubmodel.None)

	p.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	aff, _ = storage.FetchNodeAffiliation(context.Background(), "ortuman@jackal.im", "princely_musings", "noelia@jackal.im")
	require.Nil(t, aff)
}

func TestXEP163_RetrieveAffiliations(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

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

	p.ProcessIQ(context.Background(), iq)
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
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	// create node
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

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

	p.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	subs, _ := storage.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.NotNil(t, subs)
	require.Len(t, subs, 1)
	require.Equal(t, "noelia@jackal.im", subs[0].JID)
	require.Equal(t, pubsubmodel.Subscribed, subs[0].Subscription)

	// remove subscription
	sub.SetAttribute("subscription", pubsubmodel.None)

	p.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	subs, _ = storage.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, subs)
}

func TestXEP163_RetrieveSubscriptions(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        uuid.New(),
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

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

	p.ProcessIQ(context.Background(), iq)
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
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	// create node and affiliations
	nodeOpts := defaultNodeOptions
	nodeOpts.NotifySub = true

	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: nodeOpts,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	subElem := xmpp.NewElementName("subscribe")
	subElem.SetAttribute("node", "princely_musings")
	subElem.SetAttribute("jid", "noelia@jackal.im")
	pubSub.AppendElement(subElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(context.Background(), iq)
	elem := stm2.ReceiveElement()

	// command reply
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	pubSubElem := elem.Elements().ChildNamespace("pubsub", pubSubNamespace)
	require.NotNil(t, pubSubElem)
	subscriptionElem := pubSubElem.Elements().Child("subscription")
	require.NotNil(t, subscriptionElem)
	require.Equal(t, "noelia@jackal.im", subscriptionElem.Attributes().Get("jid"))
	require.Equal(t, "subscribed", subscriptionElem.Attributes().Get("subscription"))
	require.Equal(t, "princely_musings", subscriptionElem.Attributes().Get("node"))

	// subscription notification
	elem = stm1.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "message", elem.Name())

	eventElem := elem.Elements().ChildNamespace("event", pubSubEventNamespace)
	require.NotNil(t, eventElem)

	subscriptionElem = eventElem.Elements().Child("subscription")
	require.NotNil(t, subscriptionElem)
	require.Equal(t, "noelia@jackal.im", subscriptionElem.Attributes().Get("jid"))
	require.Equal(t, "subscribed", subscriptionElem.Attributes().Get("subscription"))
	require.Equal(t, "princely_musings", subscriptionElem.Attributes().Get("node"))

	// check storage subscription
	subs, _ := storage.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Len(t, subs, 1)
	require.Equal(t, "noelia@jackal.im", subs[0].JID)
	require.Equal(t, pubsubmodel.Subscribed, subs[0].Subscription)
}

func TestXEP163_Unsubscribe(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm2)

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        uuid.New(),
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	// process pubsub command
	p := New(nil, nil, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	subElem := xmpp.NewElementName("unsubscribe")
	subElem.SetAttribute("node", "princely_musings")
	subElem.SetAttribute("jid", "noelia@jackal.im")
	pubSub.AppendElement(subElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(context.Background(), iq)
	elem := stm2.ReceiveElement()

	// command reply
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// check storage subscription
	subs, _ := storage.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Len(t, subs, 0)
}

func TestXEP163_RetrieveItems(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")
	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

	// create items
	_ = storage.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID:        "i1",
		Publisher: "noelia@jackal.im",
		Payload:   xmpp.NewElementName("m1"),
	}, "ortuman@jackal.im", "princely_musings", 2)

	_ = storage.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID:        "i2",
		Publisher: "noelia@jackal.im",
		Payload:   xmpp.NewElementName("m2"),
	}, "ortuman@jackal.im", "princely_musings", 2)

	p := New(nil, nil, r)

	// retrieve all items
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1.ToBareJID())

	pubSub := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	itemsCmdElem := xmpp.NewElementName("items")
	itemsCmdElem.SetAttribute("node", "princely_musings")
	pubSub.AppendElement(itemsCmdElem)
	iq.AppendElement(pubSub)

	p.ProcessIQ(context.Background(), iq)
	elem := stm2.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	pubSubElem := elem.Elements().ChildNamespace("pubsub", pubSubNamespace)
	require.NotNil(t, pubSubElem)
	itemsElem := pubSubElem.Elements().Child("items")
	require.NotNil(t, itemsElem)
	items := itemsElem.Elements().Children("item")
	require.Len(t, items, 2)

	require.Equal(t, "i1", items[0].Attributes().Get("id"))
	require.Equal(t, "i2", items[1].Attributes().Get("id"))

	// retrieve item i2
	i2Elem := xmpp.NewElementName("item")
	i2Elem.SetAttribute("id", "i2")
	itemsCmdElem.AppendElement(i2Elem)

	p.ProcessIQ(context.Background(), iq)
	elem = stm2.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	pubSubElem = elem.Elements().ChildNamespace("pubsub", pubSubNamespace)
	require.NotNil(t, pubSubElem)
	itemsElem = pubSubElem.Elements().Child("items")
	require.NotNil(t, itemsElem)
	items = itemsElem.Elements().Children("item")
	require.Len(t, items, 1)

	require.Equal(t, "i2", items[0].Attributes().Get("id"))
}

func TestXEP163_SubscribeToAll(t *testing.T) {
	r, _ := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	// create node and affiliations
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "noelia@jackal.im",
		Name:    "princely_musings_1",
		Options: defaultNodeOptions,
	})
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "noelia@jackal.im",
		Name:    "princely_musings_2",
		Options: defaultNodeOptions,
	})
	_ = storage.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID:        "i2",
		Publisher: "noelia@jackal.im",
		Payload:   xmpp.NewElementName("m2"),
	}, "noelia@jackal.im", "princely_musings_2", 2)

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: "both",
	})
	p := New(nil, nil, r)

	err := p.subscribeToAll(context.Background(), "noelia@jackal.im", j1)
	require.Nil(t, err)

	nodes, _ := storage.FetchSubscribedNodes(context.Background(), j1.ToBareJID().String())
	require.NotNil(t, nodes)
	require.Len(t, nodes, 2)

	err = p.unsubscribeFromAll(context.Background(), "noelia@jackal.im", j1)
	require.Nil(t, err)

	nodes, _ = storage.FetchSubscribedNodes(context.Background(), j1.ToBareJID().String())
	require.Nil(t, nodes)
}

func TestXEP163_FilteredNotifications(t *testing.T) {
	r, capsRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	// create node, affiliations and subscriptions
	_ = storage.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})

	_ = storage.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: pubsubmodel.Owner,
	}, "ortuman@jackal.im", "princely_musings")

	_, _ = storage.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})

	_ = storage.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        uuid.New(),
		JID:          "noelia@jackal.im",
		Subscription: pubsubmodel.Subscribed,
	}, "ortuman@jackal.im", "princely_musings")

	// set capabilities
	_ = capsRep.InsertCapabilities(context.Background(), &model.Capabilities{
		Node:     "http://code.google.com/p/exodus",
		Ver:      "QgayPKawpkPSDYmwT/WM94uAlu0=",
		Features: []string{"princely_musings+notify"},
	})
	ph := presencehub.New(r, capsRep)

	// register presence
	pr2 := xmpp.NewPresence(j2, j2, xmpp.AvailableType)
	c := xmpp.NewElementNamespace("c", "http://jabber.org/protocol/caps")
	c.SetAttribute("hash", "sha-1")
	c.SetAttribute("node", "http://code.google.com/p/exodus")
	c.SetAttribute("ver", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	pr2.AppendElement(c)

	_, _ = ph.RegisterPresence(context.Background(), pr2)

	// process pubsub command
	p := New(nil, ph, r)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())

	pubSubEl := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	publishEl := xmpp.NewElementName("publish")
	publishEl.SetAttribute("node", "princely_musings")
	itemEl := xmpp.NewElementName("item")
	itemEl.SetAttribute("id", "bnd81g37d61f49fgn581")
	entryEl := xmpp.NewElementNamespace("entry", "http://www.w3.org/2005/Atom")
	itemEl.AppendElement(entryEl)
	publishEl.AppendElement(itemEl)
	pubSubEl.AppendElement(publishEl)

	iq.AppendElement(pubSubEl)

	p.ProcessIQ(context.Background(), iq)
	elem := stm2.ReceiveElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, xmpp.HeadlineType, elem.Type())

	eventEl := elem.Elements().ChildNamespace("event", pubSubEventNamespace)
	require.NotNil(t, eventEl)

	itemsEl := eventEl.Elements().Child("items")
	require.NotNil(t, itemsEl)

	require.Equal(t, "bnd81g37d61f49fgn581", itemsEl.Elements().Child("item").Attributes().Get("id"))
}

func setupTest(domain string) (*router.Router, *memorystorage.Capabilities) {
	// ===========================
	storage.Unset()
	s2 := memorystorage.New2()
	storage.Set(s2)
	// ===========================

	capsRep := memorystorage.NewCapabilities()
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
	}, memorystorage.NewUser())
	return r, capsRep
}
