/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"crypto/tls"
	"errors"
	"sync"

	"github.com/ortuman/jackal/version"

	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

const bindMsgBatchSize = 1024

var (
	// ErrNotExistingAccount will be returned by Route method
	// if destination user does not exist.
	ErrNotExistingAccount = errors.New("router: account does not exist")

	// ErrResourceNotFound will be returned by Route method
	// if destination resource does not match any of user's available resources.
	ErrResourceNotFound = errors.New("router: resource not found")

	// ErrNotAuthenticated will be returned by Route method if
	// destination user is not available at this moment.
	ErrNotAuthenticated = errors.New("router: user not authenticated")

	// ErrBlockedJID will be returned by Route method if
	// destination jid matches any of the user's blocked jid.
	ErrBlockedJID = errors.New("router: destination jid is blocked")

	// ErrFailedRemoteConnect will be returned by Route method if
	// couldn't establish a connection to the remote server.
	ErrFailedRemoteConnect = errors.New("router: failed remote connection")
)

// OutS2SProvider provides a specific s2s outgoing connection for every single
// pair of (localdomain, remotedomain) values.
type OutS2SProvider interface {
	GetOut(localDomain, remoteDomain string) (stream.S2SOut, error)
}

// Cluster represents the generic cluster interface used by router type.
type Cluster interface {
	// LocalNode returns local node name.
	LocalNode() string

	C2SStream(jid *jid.JID, presence *xmpp.Presence, context map[string]interface{}, node string) *cluster.C2S

	SendMessageTo(node string, message *cluster.Message)

	BroadcastMessage(msg *cluster.Message)
}

// Router represents an XMPP stanza router.
type Router struct {
	mu             sync.RWMutex
	outS2SProvider OutS2SProvider
	cluster        Cluster
	hosts          map[string]tls.Certificate
	streams        map[string][]stream.C2S
	localStreams   map[string]stream.C2S
	clusterStreams map[string]map[string]*cluster.C2S

	blockListsMu sync.RWMutex
	blockLists   map[string][]*jid.JID
}

// New returns an new empty router instance.
func New(config *Config) (*Router, error) {
	r := &Router{
		hosts:          make(map[string]tls.Certificate),
		blockLists:     make(map[string][]*jid.JID),
		streams:        make(map[string][]stream.C2S),
		localStreams:   make(map[string]stream.C2S),
		clusterStreams: make(map[string]map[string]*cluster.C2S),
	}
	if len(config.Hosts) > 0 {
		for _, h := range config.Hosts {
			r.hosts[h.Name] = h.Certificate
		}
	} else {
		cer, err := util.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return nil, err
		}
		r.hosts[defaultDomain] = cer
	}
	return r, nil
}

// HostNames returns the list of all configured host names.
func (r *Router) HostNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var ret []string
	for n := range r.hosts {
		ret = append(ret, n)
	}
	return ret
}

// IsLocalHost returns true if domain is a local server domain.
func (r *Router) IsLocalHost(domain string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.hosts[domain]
	return ok
}

// Certificates returns an array of all configured domain certificates.
func (r *Router) Certificates() []tls.Certificate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var certs []tls.Certificate
	for _, cer := range r.hosts {
		certs = append(certs, cer)
	}
	return certs
}

// SetOutS2SProvider sets the s2s out provider to be used when routing stanzas remotely.
func (r *Router) SetOutS2SProvider(provider OutS2SProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outS2SProvider = provider
}

// SetCluster sets router cluster interface.
func (r *Router) SetCluster(cluster Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cluster = cluster
}

func (r *Router) Cluster() Cluster {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cluster
}

// UpdateClusterPresence updates a presence associated to a jid in the whole cluster.
/*
func (r *Router) UpdateClusterPresence(presence *xmpp.Presence, j *jid.JID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// broadcast cluster 'presence' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUpdatePresence,
			Node: r.cluster.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID:    j,
				Stanza: presence,
			}},
		})
	}
}
*/

// ClusterDelegate returns a router cluster delegate interface.
func (r *Router) ClusterDelegate() cluster.Delegate {
	return &clusterDelegate{r: r}
}

// Bind marks a c2s stream as binded.
// An error will be returned in case no assigned resource is found.
func (r *Router) Bind(stm stream.C2S) {
	if len(stm.Resource()) == 0 {
		return
	}
	// bind stream
	r.mu.Lock()
	r.bind(stm)
	r.localStreams[stm.JID().String()] = stm
	r.mu.Unlock()

	log.Infof("binded c2s stream... (%s/%s)", stm.Username(), stm.Resource())

	// broadcast cluster 'bind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgBind,
			Node: r.cluster.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID:     stm.JID(),
				Stanza:  stm.Presence(),
				Context: stm.Context(),
			}},
		})
	}
	return
}

// Unbind unbinds a previously binded c2s.
// An error will be returned in case no assigned resource is found.
func (r *Router) Unbind(stmJID *jid.JID) {
	if len(stmJID.Resource()) == 0 {
		return
	}
	// unbind stream
	r.mu.Lock()
	if found := r.unbind(stmJID); !found {
		r.mu.Unlock()
		return
	}
	delete(r.localStreams, stmJID.String())
	r.mu.Unlock()

	log.Infof("unbinded c2s stream... (%s/%s)", stmJID.Node(), stmJID.Resource())

	// broadcast cluster 'unbind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUnbind,
			Node: r.cluster.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID: stmJID,
			}},
		})
	}
}

// UserStreams returns all streams associated to a user.
func (r *Router) UserStreams(username string) []stream.C2S {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.streams[username]
}

// IsBlockedJID returns whether or not the passed jid matches any of a user's blocking list jid.
func (r *Router) IsBlockedJID(jid *jid.JID, username string) bool {
	bl := r.getBlockList(username)
	for _, blkJID := range bl {
		if r.jidMatchesBlockedJID(jid, blkJID) {
			return true
		}
	}
	return false
}

// ReloadBlockList reloads in memory block list for a given user and starts applying it for future stanza routing.
func (r *Router) ReloadBlockList(username string) {
	r.blockListsMu.Lock()
	defer r.blockListsMu.Unlock()

	delete(r.blockLists, username)
	log.Infof("block list reloaded... (username: %s)", username)
}

// Route routes a stanza applying server rules for handling XML stanzas.
// (https://xmpp.org/rfcs/rfc3921.html#rules)
func (r *Router) Route(stanza xmpp.Stanza) error {
	return r.route(stanza, false)
}

// MustRoute routes a stanza applying server rules for handling XML stanzas
// ignoring blocking lists.
func (r *Router) MustRoute(stanza xmpp.Stanza) error {
	return r.route(stanza, true)
}

func (r *Router) jidMatchesBlockedJID(j, blockedJID *jid.JID) bool {
	if blockedJID.IsFullWithUser() {
		return j.Matches(blockedJID, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource)
	} else if blockedJID.IsFullWithServer() {
		return j.Matches(blockedJID, jid.MatchesDomain|jid.MatchesResource)
	} else if blockedJID.IsBare() {
		return j.Matches(blockedJID, jid.MatchesNode|jid.MatchesDomain)
	}
	return j.Matches(blockedJID, jid.MatchesDomain)
}

func (r *Router) getBlockList(username string) []*jid.JID {
	r.blockListsMu.RLock()
	bl := r.blockLists[username]
	r.blockListsMu.RUnlock()
	if bl != nil {
		return bl
	}
	blItems, err := storage.FetchBlockListItems(username)
	if err != nil {
		log.Error(err)
		return nil
	}
	bl = []*jid.JID{}
	for _, blItem := range blItems {
		j, _ := jid.NewWithString(blItem.JID, true)
		bl = append(bl, j)
	}
	r.blockListsMu.Lock()
	r.blockLists[username] = bl
	r.blockListsMu.Unlock()
	return bl
}

func (r *Router) bind(stm stream.C2S) {
	if usrStreams := r.streams[stm.Username()]; usrStreams != nil {
		res := stm.Resource()
		for _, usrStream := range usrStreams {
			if usrStream.Resource() == res {
				return // already binded
			}
		}
		r.streams[stm.Username()] = append(usrStreams, stm)
	} else {
		r.streams[stm.Username()] = []stream.C2S{stm}
	}
}

func (r *Router) unbind(jid *jid.JID) bool {
	found := false
	if usrStreams := r.streams[jid.Node()]; usrStreams != nil {
		res := jid.Resource()
		for i := 0; i < len(usrStreams); i++ {
			if res == usrStreams[i].Resource() {
				usrStreams = append(usrStreams[:i], usrStreams[i+1:]...)
				if len(usrStreams) > 0 {
					r.streams[jid.Node()] = usrStreams
				} else {
					delete(r.streams, jid.Node())
				}
				found = true
				break
			}
		}
	}
	return found
}

func (r *Router) route(element xmpp.Stanza, ignoreBlocking bool) error {
	toJID := element.ToJID()
	if !ignoreBlocking && !toJID.IsServer() {
		if r.IsBlockedJID(element.FromJID(), toJID.Node()) {
			return ErrBlockedJID
		}
	}
	if !r.IsLocalHost(toJID.Domain()) {
		return r.remoteRoute(element)
	}
	rcps := r.UserStreams(toJID.Node())
	if len(rcps) == 0 {
		exists, err := storage.UserExists(toJID.Node())
		if err != nil {
			return err
		}
		if exists {
			return ErrNotAuthenticated
		}
		return ErrNotExistingAccount
	}
	if toJID.IsFullWithUser() {
		for _, stm := range rcps {
			if stm.Resource() == toJID.Resource() {
				stm.SendElement(element)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch element.(type) {
	case *xmpp.Message:
		// send to highest priority stream
		stm := rcps[0]
		var highestPriority int8
		if p := stm.Presence(); p != nil {
			highestPriority = p.Priority()
		}
		for i := 1; i < len(rcps); i++ {
			rcp := rcps[i]
			if p := rcp.Presence(); p != nil && p.Priority() > highestPriority {
				stm = rcp
				highestPriority = p.Priority()
			}
		}
		stm.SendElement(element)

	default:
		// broadcast toJID all streams
		for _, stm := range rcps {
			stm.SendElement(element)
		}
	}
	return nil
}

func (r *Router) remoteRoute(elem xmpp.Stanza) error {
	if r.outS2SProvider == nil {
		return ErrFailedRemoteConnect
	}
	localDomain := elem.FromJID().Domain()
	remoteDomain := elem.ToJID().Domain()

	out, err := r.outS2SProvider.GetOut(localDomain, remoteDomain)
	if err != nil {
		log.Error(err)
		return ErrFailedRemoteConnect
	}
	out.SendElement(elem)
	return nil
}

func (r *Router) handleNotifyMessage(msg *cluster.Message) {
	switch msg.Type {
	case cluster.MsgBatchBind, cluster.MsgBind:
		r.processBindMessage(msg)
	case cluster.MsgUnbind:
		r.processUnbindMessage(msg)
	case cluster.MsgUpdatePresence:
		r.processUpdatePresenceMessage(msg)
	case cluster.MsgRouteStanza:
		r.processRouteStanzaMessage(msg)
	}
}

func (r *Router) handleNodeJoined(node *cluster.Node) {
	if r.cluster == nil {
		return
	}
	if node.Metadata.Version != version.ApplicationVersion.String() {
		log.Warnf("incompatible node version: %s (node: %s)", node.Metadata.Version, node.Name)
		return
	}
	r.mu.RLock()

	// send local JIDs in batches to the recently joined node
	i := 0
	var payloads []cluster.MessagePayload
	for _, stm := range r.localStreams {
		payloads = append(payloads, cluster.MessagePayload{
			JID:     stm.JID(),
			Stanza:  stm.Presence(),
			Context: stm.Context(),
		})
		i++
		if i == bindMsgBatchSize {
			r.cluster.SendMessageTo(node.Name, &cluster.Message{
				Type:     cluster.MsgBatchBind,
				Node:     r.cluster.LocalNode(),
				Payloads: payloads,
			})
			payloads = nil
			i = 0
		}
	}
	// send remaining ones...
	if len(payloads) > 0 {
		r.cluster.SendMessageTo(node.Name, &cluster.Message{
			Type:     cluster.MsgBatchBind,
			Node:     r.cluster.LocalNode(),
			Payloads: payloads,
		})
	}
	r.mu.RUnlock()
}

func (r *Router) handleNodeLeft(node *cluster.Node) {
	// unbind node streams
	r.mu.Lock()
	if streams := r.clusterStreams[node.Name]; streams != nil {
		for _, stm := range streams {
			r.unbind(stm.JID())
		}
	}
	delete(r.clusterStreams, node.Name)
	r.mu.Unlock()
}

func (r *Router) processBindMessage(msg *cluster.Message) {
	r.mu.Lock()
	for _, p := range msg.Payloads {
		j := p.JID
		presence, ok := p.Stanza.(*xmpp.Presence)
		if !ok {
			continue
		}
		log.Debugf("binded cluster c2s: %s", j.String())

		stm := r.cluster.C2SStream(j, presence, p.Context, msg.Node)
		r.bind(stm)
		r.registerClusterC2S(stm, msg.Node)
	}
	r.mu.Unlock()
}

func (r *Router) processUnbindMessage(msg *cluster.Message) {
	j := msg.Payloads[0].JID

	log.Debugf("unbinded cluster c2s: %s", j.String())
	r.mu.Lock()
	r.unbind(j)
	r.unregisterClusterC2S(j, msg.Node)
	r.mu.Unlock()
}

func (r *Router) processUpdateContest(msg *cluster.Message) {
	j := msg.Payloads[0].JID
	context := msg.Payloads[0].Context

	log.Debugf("updated cluster c2s context: %s\n%v", j.String(), context)

	var stm *cluster.C2S
	r.mu.RLock()
	if streams := r.clusterStreams[msg.Node]; streams != nil {
		stm = streams[j.String()]
	}
	r.mu.RUnlock()
	if stm == nil {
		return
	}
	stm.UpdateContext(context)
}

func (r *Router) processUpdatePresenceMessage(msg *cluster.Message) {
	j := msg.Payloads[0].JID
	stanza := msg.Payloads[0].Stanza

	presence, ok := stanza.(*xmpp.Presence)
	if !ok {
		return
	}
	log.Debugf("updated cluster c2s presence: %s\n%v", j.String(), presence)

	var stm *cluster.C2S
	r.mu.RLock()
	if streams := r.clusterStreams[msg.Node]; streams != nil {
		stm = streams[j.String()]
	}
	r.mu.RUnlock()
	if stm == nil {
		return
	}
	stm.SetPresence(presence)
}

func (r *Router) processRouteStanzaMessage(msg *cluster.Message) {
	j := msg.Payloads[0].JID
	stanza := msg.Payloads[0].Stanza

	log.Debugf("routing cluster stanza: %s\n%v", j.String(), stanza)
	_ = r.route(stanza, false)
}

func (r *Router) registerClusterC2S(stm *cluster.C2S, node string) {
	if streams := r.clusterStreams[node]; streams != nil {
		streams[stm.JID().String()] = stm
	} else {
		r.clusterStreams[node] = map[string]*cluster.C2S{
			stm.JID().String(): stm,
		}
	}
}

func (r *Router) unregisterClusterC2S(jid *jid.JID, node string) {
	if streams := r.clusterStreams[node]; streams != nil {
		delete(streams, jid.String())
		if len(streams) == 0 {
			delete(r.clusterStreams, node)
		}
	}
}
