/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package presencehub

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
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

// AvailablePresenceInfo contains an active presence reference along with its capabilities.
type AvailablePresenceInfo struct {
	Presence *xmpp.Presence
	Caps     *model.Capabilities
}

// PresenceHub represents global presence hub
type PresenceHub struct {
	runQueue           *runqueue.RunQueue
	router             router.GlobalRouter
	capsRep            repository.Capabilities
	availablePresences sync.Map
	capabilities       sync.Map
	activeDiscoInfo    sync.Map
}

// New returns a new presence hub instance.
func New(router router.GlobalRouter, capsRep repository.Capabilities) *PresenceHub {
	return &PresenceHub{
		runQueue: runqueue.New("presencehub"),
		router:   router,
		capsRep:  capsRep,
	}
}

// RegisterPresence keeps track of a new client presence, requesting capabilities when necessary.
func (x *PresenceHub) RegisterPresence(ctx context.Context, presence *xmpp.Presence) (alreadyRegistered bool, err error) {
	fromJID := presence.FromJID()

	// check if caps were previously cached
	if c := presence.Capabilities(); c != nil {
		capsKey := capabilitiesKey(c.Node, c.Ver)
		_, ok := x.capabilities.Load(capsKey)
		if !ok {
			caps, err := x.capsRep.FetchCapabilities(ctx, c.Node, c.Ver) // try fetching from disk
			if err != nil {
				return false, err
			}
			if caps == nil {
				x.requestCapabilities(ctx, c.Node, c.Ver, fromJID) // request capabilities
			} else {
				x.capabilities.Store(capsKey, caps) // cache capabilities
			}
		}
	}
	// store available presence
	_, loaded := x.availablePresences.LoadOrStore(fromJID, presence)
	return loaded, nil
}

// UnregisterPresence removes a presence from the hub.
func (x *PresenceHub) UnregisterPresence(_ context.Context, presence *xmpp.Presence) {
	x.availablePresences.Delete(presence.FromJID())
}

// MatchesIQ returns whether or not an IQ should be processed by the roster module.
func (x *PresenceHub) MatchesIQ(iq *xmpp.IQ) bool {
	_, ok := x.activeDiscoInfo.Load(iq.ID())
	return ok && iq.IsResult()
}

// ProcessIQ processes a roster IQ taking according actions over the associated stream.
func (x *PresenceHub) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down blocking module.
func (x *PresenceHub) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

// AvailablePresencesMatchingJID returns current online presences matching a given JID.
func (x *PresenceHub) AvailablePresencesMatchingJID(j *jid.JID) []AvailablePresenceInfo {
	var ret []AvailablePresenceInfo
	x.availablePresences.Range(func(_, value interface{}) bool {
		switch presence := value.(type) {
		case *xmpp.Presence:
			if !x.availableJIDMatchesJID(presence.FromJID(), j) {
				return true
			}
			availPresenceInfo := AvailablePresenceInfo{Presence: presence}
			if c := presence.Capabilities(); c != nil {
				if caps, _ := x.capabilities.Load(capabilitiesKey(c.Node, c.Ver)); caps != nil {
					switch caps := caps.(type) {
					case *model.Capabilities:
						availPresenceInfo.Caps = caps
					}
				}
			}
			ret = append(ret, availPresenceInfo)
		}
		return true
	})
	return ret
}

func (x *PresenceHub) processIQ(ctx context.Context, iq *xmpp.IQ) {
	// process capabilities result
	if caps := iq.Elements().ChildNamespace("query", discoInfoNamespace); caps != nil {
		if err := x.processCapabilitiesIQ(ctx, caps); err != nil {
			log.Warnf("%v", err)
		}
		return
	}
}

func (x *PresenceHub) requestCapabilities(ctx context.Context, node, ver string, userJID *jid.JID) {
	srvJID, _ := jid.NewWithString(x.router.DefaultHostName(), true)

	iqID := uuid.New()
	x.activeDiscoInfo.Store(iqID, true)

	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(srvJID)
	iq.SetToJID(userJID)

	query := xmpp.NewElementNamespace("query", discoInfoNamespace)
	query.SetAttribute("node", node+"#"+ver)
	iq.AppendElement(query)

	log.Infof("requesting capabilities... node: %s, ver: %s", node, ver)

	_ = x.router.Route(ctx, iq)
}

func (x *PresenceHub) processCapabilitiesIQ(ctx context.Context, query xmpp.XElement) error {
	var node, ver string

	nodeStr := query.Attributes().Get("node")
	ss := strings.Split(nodeStr, "#")
	if len(ss) != 2 {
		return fmt.Errorf("presencehub: wrong node format: %s", nodeStr)
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
	caps := &model.Capabilities{
		Node:     node,
		Ver:      ver,
		Features: features,
	}
	if err := x.capsRep.UpsertCapabilities(ctx, caps); err != nil { // save into disk
		return err
	}
	x.capabilities.Store(capabilitiesKey(caps.Node, caps.Ver), caps)
	return nil
}

func (x *PresenceHub) availableJIDMatchesJID(availableJID, j *jid.JID) bool {
	if j.IsFullWithUser() {
		return availableJID.MatchesWithOptions(j, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsFullWithServer() {
		return availableJID.MatchesWithOptions(j, jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsBare() {
		return availableJID.MatchesWithOptions(j, jid.MatchesNode|jid.MatchesDomain)
	}
	return availableJID.MatchesWithOptions(j, jid.MatchesDomain)
}

func capabilitiesKey(node, ver string) string {
	return node + "#" + ver
}
