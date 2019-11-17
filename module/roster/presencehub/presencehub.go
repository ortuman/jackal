package presencehub

import (
	"strings"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const (
	discoInfoNamespace = "http://jabber.org/protocol/disco#info"
)

type AvailablePresenceInfo struct {
	Presence *xmpp.Presence
	Caps     *model.Capabilities
}

type PresenceHub struct {
	router             *router.Router
	runQueue           *runqueue.RunQueue
	availablePresences sync.Map
	capabilities       sync.Map
	activeDiscoInfo    sync.Map
}

func New(router *router.Router) *PresenceHub {
	return &PresenceHub{
		router:   router,
		runQueue: runqueue.New("presencehub"),
	}
}

func (x *PresenceHub) RegisterPresence(presence *xmpp.Presence) (err error, alreadyRegistered bool) {
	fromJID := presence.FromJID()
	userJID := fromJID.ToBareJID()

	// check if caps were previously cached
	if c := presence.Capabilities(); c != nil {
		capsKey := capabilitiesKey(c.Node, c.Ver)
		_, ok := x.capabilities.Load(capsKey)
		if !ok {
			caps, err := storage.FetchCapabilities(c.Node, c.Ver) // try fetching from disk
			if err != nil {
				return err, false
			}
			if caps != nil {
				x.capabilities.Store(capsKey, caps) // cache capabilities
			} else {
				x.requestCapabilities(caps.Node, caps.Ver, userJID) // request capabilities
			}
		}
	}
	// store available presence
	_, loaded := x.availablePresences.LoadOrStore(fromJID, presence)
	return nil, loaded
}

func (x *PresenceHub) UnregisterPresence(presence *xmpp.Presence) {
	x.availablePresences.Delete(presence.FromJID())
}

// MatchesIQ returns whether or not an IQ should be processed by the roster module.
func (x *PresenceHub) MatchesIQ(iq *xmpp.IQ) bool {
	_, ok := x.activeDiscoInfo.Load(iq.ID())
	return ok && iq.IsResult()
}

// ProcessIQ processes a roster IQ taking according actions over the associated stream.
func (x *PresenceHub) ProcessIQ(iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		// process capabilities result
		if caps := iq.Elements().ChildNamespace("query", discoInfoNamespace); caps != nil {
			x.processCapabilitiesIQ(caps)
			return
		}
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
			var availPresenceInfo AvailablePresenceInfo

			availPresenceInfo.Presence = presence
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

func (x *PresenceHub) requestCapabilities(node, ver string, userJID *jid.JID) {
	srvJID, _ := jid.NewWithString(userJID.Domain(), true)

	iqID := uuid.New()
	x.activeDiscoInfo.Store(iqID, true)

	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(srvJID)
	iq.SetToJID(userJID)

	query := xmpp.NewElementNamespace("query", discoInfoNamespace)
	query.SetAttribute("node", node+"#"+ver)
	iq.AppendElement(query)

	log.Infof("requesting capabilities... node: %s, ver: %s", node, ver)

	_ = x.router.Route(iq)
}

func (x *PresenceHub) processCapabilitiesIQ(query xmpp.XElement) {
	var node, ver string

	nodeStr := query.Attributes().Get("node")
	ss := strings.Split(nodeStr, "#")
	if len(ss) != 2 {
		log.Warnf("wrong node format: %s", nodeStr)
		return
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
	if err := storage.InsertCapabilities(caps); err != nil { // save into disk
		log.Warnf("%v", err)
		return
	}
	x.capabilities.Store(capabilitiesKey(caps.Node, caps.Ver), caps)
}

func (x *PresenceHub) availableJIDMatchesJID(availableJID, j *jid.JID) bool {
	if j.IsFullWithUser() {
		return availableJID.Matches(j, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsFullWithServer() {
		return availableJID.Matches(j, jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsBare() {
		return availableJID.Matches(j, jid.MatchesNode|jid.MatchesDomain)
	}
	return availableJID.Matches(j, jid.MatchesDomain)
}

func capabilitiesKey(node, ver string) string {
	return node + "#" + ver
}
