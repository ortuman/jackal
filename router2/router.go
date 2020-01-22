/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router2

import (
	"context"
	"crypto/tls"
	"sort"

	utiltls "github.com/ortuman/jackal/util/tls"

	"github.com/ortuman/jackal/xmpp"
)

const defaultDomain = "localhost"

type Router interface {
	Route(ctx context.Context, stanza xmpp.Stanza) error
}

type GlobalRouter interface {
	Router

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

func (r *router) Route(ctx context.Context, stanza xmpp.Stanza) error {
	toJID := stanza.ToJID()
	if !r.IsLocalHost(toJID.Domain()) {
		if r.s2s == nil {
			return ErrFailedRemoteConnect
		}
		return r.s2s.Route(ctx, stanza)
	}
	return r.local.Route(ctx, stanza)
}
