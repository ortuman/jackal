/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"crypto/tls"
	"errors"
	"sync"

	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

const clusterBindBatchSize = 500

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

	C2SStream(identifier string, jid *jid.JID, node string) *cluster.C2S

	SendMessageTo(node string, msg *cluster.Message)

	BroadcastMessage(msg *cluster.Message)
}

// Router represents an XMPP stanza router.
type Router struct {
	mu             sync.RWMutex
	outS2SProvider OutS2SProvider
	cluster        Cluster
	hosts          map[string]tls.Certificate
	streams        map[string][]stream.C2S
	localJIDs      map[string]*jid.JID
	clusterStreams map[string][]*cluster.C2S

	blockListsMu sync.RWMutex
	blockLists   map[string][]*jid.JID
}

// New returns an new empty router instance.
func New(config *Config) (*Router, error) {
	r := &Router{
		hosts:          make(map[string]tls.Certificate),
		blockLists:     make(map[string][]*jid.JID),
		streams:        make(map[string][]stream.C2S),
		localJIDs:      make(map[string]*jid.JID),
		clusterStreams: make(map[string][]*cluster.C2S),
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
	for n, _ := range r.hosts {
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
func (r *Router) SetOutS2SProvider(s2sOutProvider OutS2SProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outS2SProvider = s2sOutProvider
}

// SetCluster sets router cluster interface.
func (r *Router) SetCluster(cluster Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cluster = cluster
}

// UpdateClusterPresence updates a presence associated to a jid in the whole cluster.
func (r *Router) UpdateClusterPresence(presence *xmpp.Presence, j *jid.JID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// broadcast cluster 'presence' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUpdatePresenceType,
			Node: r.cluster.LocalNode(),
			JIDs: []*jid.JID{j},
		})
	}
}

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
	defer r.mu.Unlock()

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
	r.localJIDs[stm.JID().String()] = stm.JID()

	log.Infof("binded c2s stream... (%s/%s)", stm.Username(), stm.Resource())

	// broadcast cluster 'bind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgBindType,
			Node: r.cluster.LocalNode(),
			JIDs: []*jid.JID{stm.JID()},
		})
	}
	return
}

// Unbind unbinds a previously binded c2s.
// An error will be returned in case no assigned resource is found.
func (r *Router) Unbind(stm stream.C2S) {
	if len(stm.Resource()) == 0 {
		return
	}
	// unbind stream
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	if usrStreams := r.streams[stm.Username()]; usrStreams != nil {
		res := stm.Resource()
		for i := 0; i < len(usrStreams); i++ {
			if res == usrStreams[i].Resource() {
				usrStreams = append(usrStreams[:i], usrStreams[i+1:]...)
				if len(usrStreams) > 0 {
					r.streams[stm.Username()] = usrStreams
				} else {
					delete(r.streams, stm.Username())
				}
				found = true
				break
			}
		}
	}
	if !found {
		return
	}
	delete(r.localJIDs, stm.JID().String())

	log.Infof("unbinded c2s stream... (%s/%s)", stm.Username(), stm.Resource())

	// broadcast cluster 'unbind' message
	if r.cluster != nil {
		r.cluster.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUnbindType,
			Node: r.cluster.LocalNode(),
			JIDs: []*jid.JID{stm.JID()},
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
	blItms, err := storage.FetchBlockListItems(username)
	if err != nil {
		log.Error(err)
		return nil
	}
	bl = []*jid.JID{}
	for _, blItm := range blItms {
		j, _ := jid.NewWithString(blItm.JID, true)
		bl = append(bl, j)
	}
	r.blockListsMu.Lock()
	r.blockLists[username] = bl
	r.blockListsMu.Unlock()
	return bl
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
	log.Infof("received message! %T", msg)
}

func (r *Router) handleNodeJoined(node *cluster.Node) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// generate local JIDs array
	localJIDs := make([]*jid.JID, 0, len(r.localJIDs))
	for _, j := range r.localJIDs {
		localJIDs = append(localJIDs, j)
	}
	// send local JIDs in batches to the recently joined node
	batchCount := len(localJIDs) / clusterBindBatchSize
	for i := 0; i < batchCount; i++ {
		var jBatch []*jid.JID
		for j := 0; j < clusterBindBatchSize; j++ {
			jBatch = append(jBatch, localJIDs[i*clusterBindBatchSize+j])
		}
		r.cluster.SendMessageTo(node.Name, &cluster.Message{
			Type: cluster.MsgBindType,
			Node: r.cluster.LocalNode(),
			JIDs: jBatch,
		})
	}
	log.Infof("join notified: %s", node.Name)
}

func (r *Router) handleNodeUpdated(node *cluster.Node) {
	log.Infof("update notified: %s", node.Name)
}

func (r *Router) handleNodeLeft(node *cluster.Node) {
	log.Infof("leave notified: %s", node.Name)
}
