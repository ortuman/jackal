/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"context"
	"crypto/tls"
	"runtime"
	"sort"
	"sync"

	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

var bindMsgBatchSize = 1024

// OutS2SProvider provides a specific s2s outgoing connection for every single
// pair of (localdomain, remotedomain) values.
type OutS2SProvider interface {
	GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error)
}

// Cluster represents the generic cluster interface used by router type.
type Cluster interface {
	// LocalNode returns local node name.
	LocalNode() string

	C2SStream(jid *jid.JID, presence *xmpp.Presence, context map[string]interface{}, node string) *cluster.C2S

	SendMessageTo(ctx context.Context, node string, message *cluster.Message)

	BroadcastMessage(ctx context.Context, msg *cluster.Message)
}

// Router represents an XMPP stanza router.
type Router struct {
	mu              sync.RWMutex
	outS2SProvider  OutS2SProvider
	defaultHostname string
	hosts           map[string]tls.Certificate
	streams         map[string][]stream.C2S
	cluster         Cluster
	localStreams    map[string]stream.C2S
	clusterStreams  map[string]map[string]*cluster.C2S

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
		for i, h := range config.Hosts {
			if i == 0 {
				r.defaultHostname = h.Name
			}
			r.hosts[h.Name] = h.Certificate
		}
	} else {
		cer, err := util.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return nil, err
		}
		r.defaultHostname = defaultDomain
		r.hosts[defaultDomain] = cer
	}
	return r, nil
}

// DefaultHostName returns default local host name
func (r *Router) DefaultHostName() (hostname string) {
	return r.defaultHostname
}

// HostNames returns the list of all configured host names.
func (r *Router) HostNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var ret []string
	for n := range r.hosts {
		ret = append(ret, n)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
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

// Cluster returns current router cluster.
func (r *Router) Cluster() Cluster {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cluster
}

// ClusterDelegate returns a router cluster delegate interface.
func (r *Router) ClusterDelegate() cluster.Delegate {
	return &clusterDelegate{r: r}
}

// Bind sets a c2s stream as bound.
// An error will be returned in case no assigned resource is found.
func (r *Router) Bind(ctx context.Context, stm stream.C2S) {
	if len(stm.Resource()) == 0 {
		return
	}
	// bind stream
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bind(stm)
	r.localStreams[stm.JID().String()] = stm

	log.Infof("bound c2s stream... (%s/%s)", stm.Username(), stm.Resource())

	// broadcast cluster 'bind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(ctx, &cluster.Message{
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

// Unbind unbinds a previously bound c2s stream.
// An error will be returned in case no assigned resource is found.
func (r *Router) Unbind(ctx context.Context, stmJID *jid.JID) {
	if len(stmJID.Resource()) == 0 {
		return
	}
	// unbind stream
	r.mu.Lock()
	defer r.mu.Unlock()

	if found := r.unbind(stmJID); !found {
		return
	}
	delete(r.localStreams, stmJID.String())

	log.Infof("unbound c2s stream... (%s/%s)", stmJID.Node(), stmJID.Resource())

	// broadcast cluster 'unbind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(ctx, &cluster.Message{
			Type: cluster.MsgUnbind,
			Node: r.cluster.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID: stmJID,
			}},
		})
	}
}

// UserStreams returns the stream associated to a user jid.
func (r *Router) UserStream(j *jid.JID) stream.C2S {
	r.mu.Lock()
	defer r.mu.Unlock()

	streams := r.streams[j.Node()]
	for _, stm := range streams {
		if j.Matches(stm.JID(), jid.MatchesFull) {
			return stm
		}
	}
	return nil
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
func (r *Router) Route(ctx context.Context, stanza xmpp.Stanza) error {
	return r.route(ctx, stanza, false)
}

// MustRoute routes a stanza applying server rules for handling XML stanzas
// ignoring blocking lists.
func (r *Router) MustRoute(ctx context.Context, stanza xmpp.Stanza) error {
	return r.route(ctx, stanza, true)
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
				return // already bound
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

func (r *Router) route(ctx context.Context, element xmpp.Stanza, ignoreBlocking bool) error {
	toJID := element.ToJID()
	if !ignoreBlocking && !toJID.IsServer() {
		if r.IsBlockedJID(element.FromJID(), toJID.Node()) {
			return ErrBlockedJID
		}
	}
	if !r.IsLocalHost(toJID.Domain()) {
		return r.remoteRoute(ctx, element)
	}
	recipients := r.streams[toJID.Node()]
	if len(recipients) == 0 {
		exists, err := storage.UserExists(ctx, toJID.Node())
		if err != nil {
			return err
		}
		if exists {
			return ErrNotAuthenticated
		}
		return ErrNotExistingAccount
	}
	if toJID.IsFullWithUser() {
		for _, stm := range recipients {
			if stm.Resource() == toJID.Resource() {
				stm.SendElement(ctx, element)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch element.(type) {
	case *xmpp.Message:
		// send to highest priority stream
		stm := recipients[0]
		var highestPriority int8
		if p := stm.Presence(); p != nil {
			highestPriority = p.Priority()
		}
		for i := 1; i < len(recipients); i++ {
			rcp := recipients[i]
			if p := rcp.Presence(); p != nil && p.Priority() > highestPriority {
				stm = rcp
				highestPriority = p.Priority()
			}
		}
		stm.SendElement(ctx, element)

	default:
		// broadcast toJID all streams
		for _, stm := range recipients {
			stm.SendElement(ctx, element)
		}
	}
	return nil
}

func (r *Router) remoteRoute(ctx context.Context, elem xmpp.Stanza) error {
	if r.outS2SProvider == nil {
		return ErrFailedRemoteConnect
	}
	localDomain := r.defaultHostname
	remoteDomain := elem.ToJID().Domain()

	out, err := r.outS2SProvider.GetOut(ctx, localDomain, remoteDomain)
	if err != nil {
		log.Error(err)
		return ErrFailedRemoteConnect
	}
	out.SendElement(ctx, elem)
	return nil
}

func (r *Router) handleNotifyMessage(ctx context.Context, msg *cluster.Message) {
	switch msg.Type {
	case cluster.MsgBatchBind, cluster.MsgBind:
		r.processBindMessage(ctx, msg)
	case cluster.MsgUnbind:
		r.processUnbindMessage(ctx, msg)
	case cluster.MsgUpdatePresence:
		r.processUpdatePresenceMessage(ctx, msg)
	case cluster.MsgUpdateContext:
		r.processUpdateContext(ctx, msg)
	case cluster.MsgRouteStanza:
		r.processRouteStanzaMessage(ctx, msg)
	}
}

func (r *Router) handleNodeJoined(ctx context.Context, node *cluster.Node) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cluster == nil {
		return
	}

	if node.Metadata.Version != version.ApplicationVersion.String() {
		log.Warnf("incompatible server version: %s (node: %s)", node.Metadata.Version, node.Name)
		return
	}
	if node.Metadata.GoVersion != runtime.Version() {
		log.Warnf("incompatible runtime version: %s (node: %s)", node.Metadata.GoVersion, node.Name)
		return
	}
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
			r.cluster.SendMessageTo(ctx, node.Name, &cluster.Message{
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
		r.cluster.SendMessageTo(ctx, node.Name, &cluster.Message{
			Type:     cluster.MsgBatchBind,
			Node:     r.cluster.LocalNode(),
			Payloads: payloads,
		})
	}
}

func (r *Router) handleNodeLeft(_ context.Context, node *cluster.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// unbind node streams
	if streams := r.clusterStreams[node.Name]; streams != nil {
		for _, stm := range streams {
			r.unbind(stm.JID())
		}
	}
	delete(r.clusterStreams, node.Name)
}

func (r *Router) processBindMessage(_ context.Context, msg *cluster.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cluster == nil {
		return
	}
	for _, p := range msg.Payloads {
		j := p.JID
		presence, ok := p.Stanza.(*xmpp.Presence)
		if !ok {
			continue
		}
		log.Debugf("bound cluster c2s: %s", j.String())

		stm := r.cluster.C2SStream(j, presence, p.Context, msg.Node)
		r.bind(stm)
		r.registerClusterC2S(stm, msg.Node)
	}
}

func (r *Router) processUnbindMessage(_ context.Context, msg *cluster.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cluster == nil {
		return
	}
	j := msg.Payloads[0].JID

	log.Debugf("unbound cluster c2s: %s", j.String())

	r.unbind(j)
	r.unregisterClusterC2S(j, msg.Node)
}

func (r *Router) processUpdateContext(_ context.Context, msg *cluster.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cluster == nil {
		return
	}
	j := msg.Payloads[0].JID
	stmContext := msg.Payloads[0].Context

	log.Debugf("updated cluster c2s context: %s\n%v", j.String(), stmContext)

	var stm *cluster.C2S
	if streams := r.clusterStreams[msg.Node]; streams != nil {
		stm = streams[j.String()]
	}
	if stm == nil {
		return
	}
	stm.UpdateContext(stmContext)
}

func (r *Router) processUpdatePresenceMessage(_ context.Context, msg *cluster.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cluster == nil {
		return
	}
	j := msg.Payloads[0].JID
	stanza := msg.Payloads[0].Stanza

	presence, ok := stanza.(*xmpp.Presence)
	if !ok {
		return
	}
	log.Debugf("updated cluster c2s presence: %s\n%v", j.String(), presence)

	var stm *cluster.C2S
	if streams := r.clusterStreams[msg.Node]; streams != nil {
		stm = streams[j.String()]
	}
	if stm == nil {
		return
	}
	stm.SetPresence(presence)
}

func (r *Router) processRouteStanzaMessage(ctx context.Context, msg *cluster.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cluster == nil {
		return
	}
	j := msg.Payloads[0].JID
	stanza := msg.Payloads[0].Stanza

	log.Debugf("routing cluster stanza: %s\n%v", j.String(), stanza)

	_ = r.route(ctx, stanza, false)
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
