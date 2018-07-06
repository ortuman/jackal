/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"errors"
	"sync"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
)

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
	// destination JID matches any of the user's blocked JID.
	ErrBlockedJID = errors.New("router: destination jid is blocked")

	// ErrFailedRemoteConnect will be returned by Route method if
	// couldn't establish a connection to the remote server.
	ErrFailedRemoteConnect = errors.New("router: failed remote connection")
)

type Config struct {
	GetS2SOut func(localDomain, remoteDomain string) (stream.S2SOut, error)
}

type router struct {
	cfg          *Config
	mu           sync.RWMutex
	localStreams map[string][]stream.C2S
	blockListsMu sync.RWMutex
	blockLists   map[string][]*jid.JID
}

// singleton interface
var (
	instMu      sync.RWMutex
	inst        *router
	initialized bool
)

// Initialize initializes the router manager.
func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	inst = &router{
		cfg:          cfg,
		blockLists:   make(map[string][]*jid.JID),
		localStreams: make(map[string][]stream.C2S),
	}
	initialized = true
}

// Shutdown shuts down router manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return
	}
	inst = nil
	initialized = false
}

// Bind marks a c2s stream as binded.
// An error will be returned in case no assigned resource is found.
func Bind(stm stream.C2S) {
	instance().bind(stm)
}

// Unbind unbinds a previously binded c2s.
// An error will be returned in case no assigned resource is found.
func Unbind(stm stream.C2S) {
	instance().unbind(stm)
}

// UserStreams returns all streams associated to a user.
func UserStreams(username string) []stream.C2S {
	return instance().userStreams(username)
}

// IsBlockedJID returns whether or not the passed jid matches any
// of a user's blocking list JID.
func IsBlockedJID(jid *jid.JID, username string) bool {
	return instance().isBlockedJID(jid, username)
}

// ReloadBlockList reloads in memory block list for a given user and starts
// applying it for future stanza routing.
func ReloadBlockList(username string) {
	instance().reloadBlockList(username)
}

// Route routes a stanza applying server rules for handling XML stanzas.
// (https://xmpp.org/rfcs/rfc3921.html#rules)
func Route(elem xml.Stanza) error {
	return instance().route(elem, false)
}

// MustRoute routes a stanza applying server rules for handling XML stanzas
// ignoring blocking lists.
func MustRoute(elem xml.Stanza) error {
	return instance().route(elem, true)
}

func instance() *router {
	instMu.RLock()
	defer instMu.RUnlock()
	if inst == nil {
		log.Fatalf("router manager not initialized")
	}
	return inst
}

func (r *router) bind(stm stream.C2S) {
	if len(stm.Resource()) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if authenticated := r.localStreams[stm.Username()]; authenticated != nil {
		r.localStreams[stm.Username()] = append(authenticated, stm)
	} else {
		r.localStreams[stm.Username()] = []stream.C2S{stm}
	}
	log.Infof("binded c2s stream... (%s/%s)", stm.Username(), stm.Resource())
	return
}

func (r *router) unbind(stm stream.C2S) {
	if len(stm.Resource()) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if resources := r.localStreams[stm.Username()]; resources != nil {
		res := stm.Resource()
		for i := 0; i < len(resources); i++ {
			if res == resources[i].Resource() {
				resources = append(resources[:i], resources[i+1:]...)
				break
			}
		}
		if len(resources) > 0 {
			r.localStreams[stm.Username()] = resources
		} else {
			delete(r.localStreams, stm.Username())
		}
	}
	log.Infof("unbinded c2s stream... (%s/%s)", stm.Username(), stm.Resource())
}

func (r *router) userStreams(username string) []stream.C2S {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.localStreams[username]
}

func (r *router) isBlockedJID(jid *jid.JID, username string) bool {
	bl := r.getBlockList(username)
	for _, blkJID := range bl {
		if r.jidMatchesBlockedJID(jid, blkJID) {
			return true
		}
	}
	return false
}

func (r *router) jidMatchesBlockedJID(j, blockedJID *jid.JID) bool {
	if blockedJID.IsFullWithUser() {
		return j.Matches(blockedJID, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource)
	} else if blockedJID.IsFullWithServer() {
		return j.Matches(blockedJID, jid.MatchesDomain|jid.MatchesResource)
	} else if blockedJID.IsBare() {
		return j.Matches(blockedJID, jid.MatchesNode|jid.MatchesDomain)
	}
	return j.Matches(blockedJID, jid.MatchesDomain)
}

func (r *router) reloadBlockList(username string) {
	r.blockListsMu.Lock()
	defer r.blockListsMu.Unlock()

	delete(r.blockLists, username)
	log.Infof("block list reloaded... (username: %s)", username)
}

func (r *router) getBlockList(username string) []*jid.JID {
	r.blockListsMu.RLock()
	bl := r.blockLists[username]
	r.blockListsMu.RUnlock()
	if bl != nil {
		return bl
	}
	blItms, err := storage.Instance().FetchBlockListItems(username)
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

func (r *router) route(stanza xml.Stanza, ignoreBlocking bool) error {
	toJID := stanza.ToJID()
	if !ignoreBlocking && !toJID.IsServer() {
		if r.isBlockedJID(stanza.FromJID(), toJID.Node()) {
			return ErrBlockedJID
		}
	}
	if !host.IsLocalHost(toJID.Domain()) {
		return r.remoteRoute(stanza)
	}
	rcps := r.userStreams(toJID.Node())
	if len(rcps) == 0 {
		exists, err := storage.Instance().UserExists(toJID.Node())
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
				stm.SendElement(stanza)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch stanza.(type) {
	case *xml.Message:
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
		stm.SendElement(stanza)

	default:
		// broadcast toJID all streams
		for _, stm := range rcps {
			stm.SendElement(stanza)
		}
	}
	return nil
}

func (r *router) remoteRoute(stanza xml.Stanza) error {
	if r.cfg.GetS2SOut == nil {
		return ErrFailedRemoteConnect
	}
	localDomain := stanza.FromJID().Domain()
	remoteDomain := stanza.ToJID().Domain()

	out, err := r.cfg.GetS2SOut(localDomain, remoteDomain)
	if err != nil {
		log.Error(err)
		return ErrFailedRemoteConnect
	}
	out.SendElement(stanza)
	return nil
}
