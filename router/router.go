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

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
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

// C2S represents a client-to-server XMPP stream.
type C2S interface {
	ID() string

	Context() *Context

	Username() string
	Domain() string
	Resource() string

	JID() *xml.JID

	IsSecured() bool
	IsAuthenticated() bool
	IsCompressed() bool

	Presence() *xml.Presence

	SendElement(element xml.XElement)
	Disconnect(err error)
}

// Config represents a client-to-server manager configuration.
type Config struct {
	Domains []string `yaml:"domains"`
}

// Router manages the sessions associated with an account.
type Router struct {
	cfg        *Config
	lock       sync.RWMutex
	stms       map[string]C2S
	authedStms map[string][]C2S
	blockLists map[string][]*xml.JID
}

// singleton interface
var (
	inst        *Router
	instMu      sync.RWMutex
	initialized uint32
)

// Initialize initializes the c2s session manager.
func Initialize(cfg *Config) {
	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		instMu.Lock()
		defer instMu.Unlock()

		if len(cfg.Domains) == 0 {
			log.Fatalf("router: no domain specified")
		}
		inst = &Router{
			cfg:        cfg,
			stms:       make(map[string]C2S),
			authedStms: make(map[string][]C2S),
			blockLists: make(map[string][]*xml.JID),
		}
	}
}

// Instance returns the c2s session manager instance.
func Instance() *Router {
	instMu.RLock()
	defer instMu.RUnlock()

	if inst == nil {
		log.Fatalf("c2s manager not initialized")
	}
	return inst
}

// Shutdown shuts down c2s manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		instMu.Lock()
		defer instMu.Unlock()
		inst = nil
	}
}

// DefaultLocalDomain returns default local domain.
func (r *Router) DefaultLocalDomain() string {
	return r.cfg.Domains[0]
}

// IsLocalDomain returns true if domain is a local server domain.
func (r *Router) IsLocalDomain(domain string) bool {
	for _, localDomain := range r.cfg.Domains {
		if localDomain == domain {
			return true
		}
	}
	return false
}

// RegisterStream registers the specified client stream.
// An error will be returned in case the stream has been previously registered.
func (r *Router) RegisterStream(stm C2S) error {
	if !r.IsLocalDomain(stm.Domain()) {
		return fmt.Errorf("invalid domain: %s", stm.Domain())
	}
	r.lock.Lock()
	_, ok := r.stms[stm.ID()]
	if ok {
		r.lock.Unlock()
		return fmt.Errorf("stream already registered: %s", stm.ID())
	}
	r.stms[stm.ID()] = stm
	r.lock.Unlock()
	log.Infof("registered stream... (id: %s)", stm.ID())
	return nil
}

// UnregisterStream unregisters the specified client stream removing
// associated resource from the manager.
// An error will be returned in case the stream has not been previously registered.
func (r *Router) UnregisterStream(stm C2S) error {
	r.lock.Lock()
	_, ok := r.stms[stm.ID()]
	if !ok {
		r.lock.Unlock()
		return fmt.Errorf("stream not found: %s", stm.ID())
	}
	if authedStms := r.authedStms[stm.Username()]; authedStms != nil {
		res := stm.Resource()
		for i := 0; i < len(authedStms); i++ {
			if res == authedStms[i].Resource() {
				authedStms = append(authedStms[:i], authedStms[i+1:]...)
				break
			}
		}
		if len(authedStms) > 0 {
			r.authedStms[stm.Username()] = authedStms
		} else {
			delete(r.authedStms, stm.Username())
		}
	}
	delete(r.stms, stm.ID())
	r.lock.Unlock()
	log.Infof("unregistered stream... (id: %s)", stm.ID())
	return nil
}

// AuthenticateStream sets a previously registered stream as authenticated.
// An error will be returned in case no assigned resource is found.
func (r *Router) AuthenticateStream(stm C2S) error {
	if len(stm.Resource()) == 0 {
		return fmt.Errorf("resource not yet assigned: %s", stm.ID())
	}
	r.lock.Lock()
	if authedStrms := r.authedStms[stm.Username()]; authedStrms != nil {
		r.authedStms[stm.Username()] = append(authedStrms, stm)
	} else {
		r.authedStms[stm.Username()] = []C2S{stm}
	}
	r.lock.Unlock()
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

// ReloadBlockList reloads in-memstorage block list for a given user and starts
// applying it for future stanza routing.
func (r *Router) ReloadBlockList(username string) {
	r.lock.Lock()
	delete(r.blockLists, username)
	r.lock.Unlock()
	log.Infof("block list reloaded... (username: %s)", username)
}

// Route routes a stanza applying server rules for handling XML stanzas.
// (https://xmpp.org/rfcs/rfc3921.html#rules)
func (r *Router) Route(elem xml.Stanza) error {
	return r.route(elem, false)
}

// MustRoute routes a stanza applying server rules for handling XML stanzas
// and ignoring blocking lists.
func (r *Router) MustRoute(elem xml.Stanza) error {
	return r.route(elem, true)
}

// StreamsMatchingJID returns all available streams that match a given JID.
func (r *Router) StreamsMatchingJID(jid *xml.JID) []C2S {
	if !r.IsLocalDomain(jid.Domain()) {
		return nil
	}
	var ret []C2S
	opts := xml.JIDMatchesDomain
	if jid.IsFull() {
		opts |= xml.JIDMatchesResource
	}

	r.lock.RLock()
	if len(jid.Node()) > 0 {
		opts |= xml.JIDMatchesNode
		stms := r.authedStms[jid.Node()]
		for _, stm := range stms {
			if stm.JID().Matches(jid, opts) {
				ret = append(ret, stm)
			}
		}
	} else {
		for _, stms := range r.authedStms {
			for _, stm := range stms {
				if stm.JID().Matches(jid, opts) {
					ret = append(ret, stm)
				}
			}
		}
	}
	r.lock.RUnlock()
	return ret
}

func (r *Router) route(elem xml.Stanza, ignoreBlocking bool) error {
	toJID := elem.ToJID()
	if !r.IsLocalDomain(toJID.Domain()) {
		return nil
	}
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

func (r *Router) getBlockList(username string) []*xml.JID {
	r.lock.RLock()
	bl := r.blockLists[username]
	r.lock.RUnlock()
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
	r.lock.Lock()
	r.blockLists[username] = bl
	r.lock.Unlock()
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
