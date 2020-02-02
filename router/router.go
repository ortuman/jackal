/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"context"
	"crypto/tls"
	"sort"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

type Type int

const (
	// Global represents a global router type.
	Global = Type(iota)

	// Local represents a C2S router type.
	Local

	// Cluster represents a cluster router type.
	Cluster

	// Remote represents a S2S router type.
	Remote
)

type Router interface {
	// Type returns router type.
	Type() Type

	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza xmpp.Stanza) error
}

type GlobalRouter interface {
	Router

	// MustRoute forces stanza routing by ignoring user's blocking list.
	MustRoute(ctx context.Context, stanza xmpp.Stanza) error

	// Bind sets a c2s stream as bound.
	Bind(ctx context.Context, stm stream.C2S)

	// Unbind unbinds a previously bound c2s stream.
	Unbind(ctx context.Context, j *jid.JID)

	// LocalStreams returns all streams associated to a given username.
	LocalStreams(username string) []stream.C2S

	// LocalStream returns the stream associated to a given username and resource.
	LocalStream(username, resource string) stream.C2S

	// DefaultHostName returns default local host name.s
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
	s2s             Router
	blockListRep    repository.BlockList
}

func New(config *Config, userRep repository.User, blockListRep repository.BlockList) (GlobalRouter, error) {
	r := &router{
		hosts:        make(map[string]tls.Certificate),
		blockListRep: blockListRep,
	}
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
	r.local = newLocalRouter(userRep)
	return r, nil
}

func (r *router) Type() Type {
	return Global
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

func (r *router) Bind(ctx context.Context, stm stream.C2S) {
	r.local.bind(stm)
}

func (r *router) Unbind(ctx context.Context, j *jid.JID) {
	r.local.unbind(j.Node(), j.Resource())
}

func (r *router) LocalStreams(username string) []stream.C2S {
	return r.local.userStreams(username)
}

func (r *router) LocalStream(username, resource string) stream.C2S {
	return r.local.userStream(username, resource)
}

func (r *router) route(ctx context.Context, stanza xmpp.Stanza, ignoreBlocking bool) error {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	// check if sender JID is blocked
	if r.IsLocalHost(toJID.Domain()) && !ignoreBlocking {
		if r.isBlockedJID(ctx, fromJID, toJID.Node()) {
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
	if len(blockList) == 0 {
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
