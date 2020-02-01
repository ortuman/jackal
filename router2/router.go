/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router2

import (
	"context"
	"crypto/tls"
	"sort"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/repository"
	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

type Router interface {
	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza xmpp.Stanza) error
}

type GlobalRouter interface {
	Router

	// MustRoute forces stanza routing by ignoring user's blocking list.
	MustRoute(ctx context.Context, stanza xmpp.Stanza) error

	// DefaultHostName returns default local host name.
	DefaultHostName() string

	// IsLocalHost returns true if domain is a local server domain.
	IsLocalHost(domain string) bool

	// HostNames returns the list of all configured host names.
	HostNames() []string

	// Certificates returns an array of all configured domain certificates.
	Certificates() []tls.Certificate
}

type router struct {
	defaultHostname string
	hosts           map[string]tls.Certificate
	local           *localRouter
	s2s             *s2sRouter
	blockListRep    repository.BlockList
}

func New(config Config) (GlobalRouter, error) {
	r := &router{}
	if len(config.Hosts) > 0 {
		for i, h := range config.Hosts {
			if i == 0 {
				r.defaultHostname = h.Name
			}
			r.hosts[h.Name] = h.Certificate
		}
	} else {
		cer, err := utiltls.LoadCertificate("", "", defaultDomain)
		if err != nil {
			return nil, err
		}
		r.defaultHostname = defaultDomain
		r.hosts[defaultDomain] = cer
	}
	return r, nil
}

func (r *router) DefaultHostName() string {
	return r.defaultHostname
}

func (r *router) IsLocalHost(domain string) bool {
	_, ok := r.hosts[domain]
	return ok
}

func (r *router) HostNames() []string {
	var ret []string
	for n := range r.hosts {
		ret = append(ret, n)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func (r *router) Certificates() []tls.Certificate {
	var certs []tls.Certificate
	for _, cer := range r.hosts {
		certs = append(certs, cer)
	}
	return certs
}

func (r *router) MustRoute(ctx context.Context, stanza xmpp.Stanza) error {
	return r.route(ctx, stanza, true)
}

func (r *router) Route(ctx context.Context, stanza xmpp.Stanza) error {
	return r.route(ctx, stanza, false)
}

func (r *router) route(ctx context.Context, stanza xmpp.Stanza, ignoreBlocking bool) error {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	// check blocking list
	if r.IsLocalHost(fromJID.Domain()) && !ignoreBlocking {
		if r.isBlockedJID(ctx, stanza.ToJID(), fromJID.Node()) {
			return ErrBlockedJID
		}
	}
	if !r.IsLocalHost(toJID.Domain()) {
		if r.s2s == nil {
			return ErrFailedRemoteConnect
		}
		return r.s2s.Route(ctx, stanza)
	}
	return r.local.Route(ctx, stanza)
}

func (r *router) isBlockedJID(ctx context.Context, j *jid.JID, username string) bool {
	blockList, err := r.blockListRep.FetchBlockListItems(ctx, username)
	if err != nil {
		log.Error(err)
		return false
	}
	blockListJIDs := make([]jid.JID, len(blockList))
	for i, listItem := range blockList {
		j, _ := jid.NewWithString(listItem.JID, true)
		blockListJIDs[i] = *j
	}
	for _, blockedJID := range blockListJIDs {
		if blockedJID.Matches(j) {
			return true
		}
	}
	return false
}
