/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"context"
	"crypto/tls"
	"sort"

	"github.com/ortuman/jackal/stream"
	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDomain = "localhost"

type Router interface {

	// DefaultHostName returns default local host name.s
	DefaultHostName() string

	// IsLocalHost returns true if domain is a local server domain.
	IsLocalHost(domain string) bool

	// HostNames returns the list of all configured host names.
	HostNames() []string

	// Certificates returns an array of all configured domain certificates.
	Certificates() []tls.Certificate

	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza xmpp.Stanza) error

	// MustRoute forces stanza routing by ignoring user's blocking list.
	MustRoute(ctx context.Context, stanza xmpp.Stanza) error

	// Bind sets a c2s stream as bound.
	Bind(ctx context.Context, stm stream.C2S)

	// Unbind unbinds a previously bound c2s stream.
	Unbind(ctx context.Context, j *jid.JID)

	// LocalStream returns the stream associated to a given username and resource.
	LocalStream(username, resource string) stream.C2S

	// LocalStreams returns all streams associated to a given username.
	LocalStreams(username string) []stream.C2S
}

type C2SRouter interface {
	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza xmpp.Stanza, validateStanza bool) error

	// Bind sets a c2s stream as bound.
	Bind(stm stream.C2S)

	// Unbind unbinds a previously bound c2s stream.
	Unbind(username, resource string)

	// Stream returns the stream associated to a given username and resource.
	Stream(username, resource string) stream.C2S

	// Streams returns all streams associated to a given username.
	Streams(username string) []stream.C2S
}

type S2SRouter interface {
	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza xmpp.Stanza) error
}

type router struct {
	defaultHostname string
	hosts           map[string]tls.Certificate
	c2s             C2SRouter
	s2s             S2SRouter
}

func New(config *Config, c2sRouter C2SRouter, s2sRouter S2SRouter) (Router, error) {
	r := &router{
		hosts: make(map[string]tls.Certificate),
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
	r.c2s = c2sRouter
	r.s2s = s2sRouter
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
	return r.route(ctx, stanza, false)
}

func (r *router) Route(ctx context.Context, stanza xmpp.Stanza) error {
	return r.route(ctx, stanza, true)
}

func (r *router) Bind(ctx context.Context, stm stream.C2S) {
	r.c2s.Bind(stm)
}

func (r *router) Unbind(ctx context.Context, j *jid.JID) {
	r.c2s.Unbind(j.Node(), j.Resource())
}

func (r *router) LocalStreams(username string) []stream.C2S {
	return r.c2s.Streams(username)
}

func (r *router) LocalStream(username, resource string) stream.C2S {
	return r.c2s.Stream(username, resource)
}

func (r *router) route(ctx context.Context, stanza xmpp.Stanza, validateStanza bool) error {
	toJID := stanza.ToJID()
	if !r.IsLocalHost(toJID.Domain()) {
		if r.s2s == nil {
			return ErrFailedRemoteConnect
		}
		return r.s2s.Route(ctx, stanza)
	}
	return r.c2s.Route(ctx, stanza, validateStanza)
}
