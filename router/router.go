/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
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
)

// Config represents a router manager configuration.
type Config struct {
	Domains     []string `yaml:"domains"`
	S2SDisabled bool     `yaml:"s2s_disabled"`
}

// Router manages the sessions associated with an account.
type Router struct {
	mu            sync.RWMutex
	cfg           *Config
	c2sMu         sync.RWMutex
	streams       map[string]stream.C2S
	authenticated map[string][]stream.C2S
	s2sMu         sync.RWMutex
	dialer        stream.S2SDialer
	outStreams    map[string]stream.S2SOut
	blockListsMu  sync.RWMutex
	blockLists    map[string][]*xml.JID
}

// singleton interface
var (
	inst        *Router
	instMu      sync.RWMutex
	initialized uint32
)

// Initialize initializes the router manager.
func Initialize(cfg *Config, dialer stream.S2SDialer) {
	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		instMu.Lock()
		defer instMu.Unlock()

		if len(cfg.Domains) == 0 {
			log.Fatalf("no domain specified")
		}
		inst = &Router{
			cfg:           cfg,
			streams:       make(map[string]stream.C2S),
			authenticated: make(map[string][]stream.C2S),
			dialer:        dialer,
			outStreams:    make(map[string]stream.S2SOut),
			blockLists:    make(map[string][]*xml.JID),
		}
	}
}

// Instance returns the router manager instance.
func Instance() *Router {
	instMu.RLock()
	defer instMu.RUnlock()

	if inst == nil {
		log.Fatalf("router manager not initialized")
	}
	return inst
}

// Shutdown shuts down router manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		var wg sync.WaitGroup
		inst.c2sMu.RLock()
		for _, stm := range inst.streams {
			wg.Add(1)
			go func(s stream.C2S) {
				s.Disconnect(streamerror.ErrSystemShutdown)
				wg.Done()
			}(stm)
		}
		inst.c2sMu.RUnlock()

		wg.Wait()

		instMu.Lock()
		inst = nil
		instMu.Unlock()
	}
}

// DefaultLocalDomain returns default local domain.
func (r *Router) DefaultLocalDomain() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg.Domains[0]
}

// IsLocalDomain returns true if domain is a local server domain.
func (r *Router) IsLocalDomain(domain string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, localDomain := range r.cfg.Domains {
		if localDomain == domain {
			return true
		}
	}
	return false
}

// RegisterC2S registers the specified c2s stream.
// An error will be returned in case the stream has been previously registered.
func (r *Router) RegisterC2S(stm stream.C2S) error {
	if !r.IsLocalDomain(stm.Domain()) {
		return fmt.Errorf("invalid domain: %s", stm.Domain())
	}
	r.c2sMu.Lock()
	defer r.c2sMu.Unlock()

	_, ok := r.streams[stm.ID()]
	if ok {
		return fmt.Errorf("stream already registered: %s", stm.ID())
	}
	r.streams[stm.ID()] = stm
	log.Infof("registered stream... (id: %s)", stm.ID())
	return nil
}

// UnregisterC2S unregisters the specified c2s stream removing
// associated resource from the manager.
// An error will be returned in case the stream has not been previously registered.
func (r *Router) UnregisterC2S(stm stream.C2S) error {
	r.c2sMu.Lock()
	defer r.c2sMu.Unlock()

	_, ok := r.streams[stm.ID()]
	if !ok {
		return fmt.Errorf("stream not found: %s", stm.ID())
	}
	if authenticated := r.authenticated[stm.Username()]; authenticated != nil {
		res := stm.Resource()
		for i := 0; i < len(authenticated); i++ {
			if res == authenticated[i].Resource() {
				authenticated = append(authenticated[:i], authenticated[i+1:]...)
				break
			}
		}
		if len(authenticated) > 0 {
			r.authenticated[stm.Username()] = authenticated
		} else {
			delete(r.authenticated, stm.Username())
		}
	}
	delete(r.streams, stm.ID())
	log.Infof("unregistered stream... (id: %s)", stm.ID())
	return nil
}

// AuthenticateC2S marks a previously registered c2s stream as authenticated.
// An error will be returned in case no assigned resource is found.
func (r *Router) AuthenticateC2S(stm stream.C2S) error {
	if len(stm.Resource()) == 0 {
		return fmt.Errorf("resource not yet assigned: %s", stm.ID())
	}
	r.c2sMu.Lock()
	defer r.c2sMu.Unlock()

	if authenticated := r.authenticated[stm.Username()]; authenticated != nil {
		r.authenticated[stm.Username()] = append(authenticated, stm)
	} else {
		r.authenticated[stm.Username()] = []stream.C2S{stm}
	}
	log.Infof("authenticated stream... (%s/%s)", stm.Username(), stm.Resource())
	return nil
}

// IsBlockedJID returns whether or not the passed jid matches any
// of a user's blocking list JID.
func (r *Router) IsBlockedJID(jid *xml.JID, username string) bool {
	bl := r.getBlockList(username)
	for _, blkJID := range bl {
		if r.jidMatchesBlockedJID(jid, blkJID) {
			return true
		}
	}
	return false
}

// ReloadBlockList reloads in memory block list for a given user and starts
// applying it for future stanza routing.
func (r *Router) ReloadBlockList(username string) {
	r.blockListsMu.Lock()
	defer r.blockListsMu.Unlock()

	delete(r.blockLists, username)
	log.Infof("block list reloaded... (username: %s)", username)
}

// Route routes a stanza applying server rules for handling XML stanzas.
// (https://xmpp.org/rfcs/rfc3921.html#rules)
func (r *Router) Route(elem xml.Stanza) error {
	return r.route(elem, false)
}

// MustRoute routes a stanza applying server rules for handling XML stanzas
// ignoring blocking lists.
func (r *Router) MustRoute(elem xml.Stanza) error {
	return r.route(elem, true)
}

// StreamsMatchingJID returns all available c2s streams matching a given JID.
func (r *Router) StreamsMatchingJID(jid *xml.JID) []stream.C2S {
	if !r.IsLocalDomain(jid.Domain()) {
		return nil
	}

	var ret []stream.C2S
	opts := xml.JIDMatchesDomain
	if jid.IsFull() {
		opts |= xml.JIDMatchesResource
	}
	r.c2sMu.RLock()
	defer r.c2sMu.RUnlock()

	if len(jid.Node()) > 0 {
		opts |= xml.JIDMatchesNode
		stms := r.authenticated[jid.Node()]
		for _, stm := range stms {
			if stm.JID().Matches(jid, opts) {
				ret = append(ret, stm)
			}
		}
	} else {
		for _, stms := range r.authenticated {
			for _, stm := range stms {
				if stm.JID().Matches(jid, opts) {
					ret = append(ret, stm)
				}
			}
		}
	}
	return ret
}

func (r *Router) route(elem xml.Stanza, ignoreBlocking bool) error {
	toJID := elem.ToJID()
	if !r.IsLocalDomain(toJID.Domain()) {
		return r.s2sRoute(elem, ignoreBlocking)
	}
	return r.c2sRoute(elem, ignoreBlocking)
}

func (r *Router) c2sRoute(elem xml.Stanza, ignoreBlocking bool) error {
	toJID := elem.ToJID()
	if !ignoreBlocking && !toJID.IsServer() {
		if r.IsBlockedJID(elem.FromJID(), toJID.Node()) {
			return ErrBlockedJID
		}
	}
	rcps := r.StreamsMatchingJID(toJID.ToBareJID())
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
				stm.SendElement(elem)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch elem.(type) {
	case *xml.Message:
		// send toJID highest priority stream
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
		stm.SendElement(elem)

	default:
		// broadcast toJID all streams
		for _, stm := range rcps {
			stm.SendElement(elem)
		}
	}
	return nil
}

func (r *Router) s2sRoute(elem xml.Stanza, ignoreBlocking bool) error {
	return nil
}

func (r *Router) getBlockList(username string) []*xml.JID {
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
	bl = []*xml.JID{}
	for _, blItm := range blItms {
		j, _ := xml.NewJIDString(blItm.JID, true)
		bl = append(bl, j)
	}
	r.blockListsMu.Lock()
	r.blockLists[username] = bl
	r.blockListsMu.Unlock()
	return bl
}

func (r *Router) jidMatchesBlockedJID(jid, blockedJID *xml.JID) bool {
	if blockedJID.IsFullWithUser() {
		return jid.Matches(blockedJID, xml.JIDMatchesNode|xml.JIDMatchesDomain|xml.JIDMatchesResource)
	} else if blockedJID.IsFullWithServer() {
		return jid.Matches(blockedJID, xml.JIDMatchesDomain|xml.JIDMatchesResource)
	} else if blockedJID.IsBare() {
		return jid.Matches(blockedJID, xml.JIDMatchesNode|xml.JIDMatchesDomain)
	}
	return jid.Matches(blockedJID, xml.JIDMatchesDomain)
}
