/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0115

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ortuman/jackal/log"
	capsmodel "github.com/ortuman/jackal/model/capabilities"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const (
	discoInfoNamespace = "http://jabber.org/protocol/disco#info"
)

// EntityCaps represents global entity capabilities module
type EntityCaps struct {
	allocationID    string
	runQueue        *runqueue.RunQueue
	router          router.Router
	presencesRep    repository.Presences
	mu              sync.RWMutex
	activeDiscoInfo map[string]bool
}

// New returns a new presence hub instance.
func New(router router.Router, presencesRep repository.Presences, allocationID string) *EntityCaps {
	return &EntityCaps{
		runQueue:        runqueue.New("xep0115"),
		router:          router,
		presencesRep:    presencesRep,
		allocationID:    allocationID,
		activeDiscoInfo: make(map[string]bool),
	}
}

// RegisterPresence keeps track of a new client presence, requesting capabilities when necessary.
func (x *EntityCaps) RegisterPresence(ctx context.Context, presence *xmpp.Presence) (alreadyRegistered bool, err error) {
	fromJID := presence.FromJID()

	// check if caps were previously cached
	if c := presence.Capabilities(); c != nil {
		if err := x.registerCapabilities(ctx, c.Node, c.Ver, presence.FromJID()); err != nil {
			return false, err
		}
	}
	// store available presence
	inserted, err := x.presencesRep.UpsertPresence(ctx, presence, fromJID, x.allocationID)
	if err != nil {
		return false, err
	}
	return inserted, nil
}

// UnregisterPresence removes a presence from the hub.
func (x *EntityCaps) UnregisterPresence(ctx context.Context, jid *jid.JID) error {
	return x.presencesRep.DeletePresence(ctx, jid)
}

// MatchesIQ returns whether or not an IQ should be processed by the roster module.
func (x *EntityCaps) MatchesIQ(iq *xmpp.IQ) bool {
	x.mu.RLock()
	defer x.mu.RUnlock()
	_, ok := x.activeDiscoInfo[iq.ID()]
	return ok && iq.IsResult()
}

// ProcessIQ processes a roster IQ taking according actions over the associated stream.
func (x *EntityCaps) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down blocking module.
func (x *EntityCaps) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

// PresencesMatchingJID returns current online presences matching a given JID.
func (x *EntityCaps) PresencesMatchingJID(ctx context.Context, jid *jid.JID) ([]capsmodel.PresenceCaps, error) {
	return x.presencesRep.FetchPresencesMatchingJID(ctx, jid)
}

func (x *EntityCaps) registerCapabilities(ctx context.Context, node, ver string, jid *jid.JID) error {
	caps, err := x.presencesRep.FetchCapabilities(ctx, node, ver) // try fetching from disk
	if err != nil {
		return err
	}
	if caps == nil {
		x.requestCapabilities(ctx, node, ver, jid) // request capabilities
	}
	return nil
}

func (x *EntityCaps) processIQ(ctx context.Context, iq *xmpp.IQ) {
	caps := iq.Elements().ChildNamespace("query", discoInfoNamespace)
	if caps == nil {
		return
	}
	// process capabilities result
	if err := x.processCapabilitiesIQ(ctx, caps); err != nil {
		log.Warnf("%v", err)
	}
}

func (x *EntityCaps) requestCapabilities(ctx context.Context, node, ver string, userJID *jid.JID) {
	srvJID, _ := jid.NewWithString(x.router.Hosts().DefaultHostName(), true)

	iqID := uuid.New()
	x.mu.Lock()
	x.activeDiscoInfo[iqID] = true
	x.mu.Unlock()

	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(srvJID)
	iq.SetToJID(userJID)

	query := xmpp.NewElementNamespace("query", discoInfoNamespace)
	query.SetAttribute("node", node+"#"+ver)
	iq.AppendElement(query)

	log.Infof("requesting capabilities... node: %s, ver: %s", node, ver)

	_ = x.router.Route(ctx, iq)
}

func (x *EntityCaps) processCapabilitiesIQ(ctx context.Context, query xmpp.XElement) error {
	var node, ver string

	nodeStr := query.Attributes().Get("node")
	ss := strings.Split(nodeStr, "#")
	if len(ss) != 2 {
		return fmt.Errorf("xep0115: wrong node format: %s", nodeStr)
	}
	node = ss[0]
	ver = ss[1]

	// retrieve and store features
	log.Infof("storing capabilities... node: %s, ver: %s", node, ver)

	var features []string
	featureElems := query.Elements().Children("feature")
	for _, featureElem := range featureElems {
		features = append(features, featureElem.Attributes().Get("var"))
	}
	caps := &capsmodel.Capabilities{
		Node:     node,
		Ver:      ver,
		Features: features,
	}
	return x.presencesRep.UpsertCapabilities(ctx, caps)
}
