// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/host"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/router/stream"
)

// Router defines global router interface.
type Router interface {

	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza stravaganza.Stanza) (targets []jid.JID, err error)

	// C2S returns the underlying C2S router.
	C2S() C2SRouter

	// S2S returns the underlying S2S router.
	S2S() S2SRouter

	// Start starts global router subsystem.
	Start(ctx context.Context) error

	// Stop stops global router subsystem.
	Stop(ctx context.Context) error
}

// RoutingOptions represents C2S routing options mask.
type RoutingOptions int8

const (
	// CheckUserExistence tells whether to check if the recipient user exists.
	CheckUserExistence = RoutingOptions(1 << 0)
)

// C2SRouter defines C2S router interface.
type C2SRouter interface {
	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza stravaganza.Stanza, routingOpts RoutingOptions) (targets []jid.JID, err error)

	// Disconnect performs disconnection over an available resource.
	Disconnect(ctx context.Context, res c2smodel.ResourceDesc, streamErr *streamerror.Error) error

	// Register registers a new stream.
	Register(stm stream.C2S) error

	// Bind sets a previously registered stream as bounded.
	Bind(id stream.C2SID) error

	// Unregister unregisters a stream.
	Unregister(stm stream.C2S) error

	// LocalStream returns local instance stream.
	LocalStream(username, resource string) (stream.C2S, error)

	// Start starts C2S router subsystem.
	Start(ctx context.Context) error

	// Stop stops C2S router subsystem.
	Stop(ctx context.Context) error
}

// S2SRouter defines S2S router interface.
type S2SRouter interface {
	// Route routes a stanza applying server rules for handling XML stanzas.
	// (https://xmpp.org/rfcs/rfc3921.html#rules)
	Route(ctx context.Context, stanza stravaganza.Stanza, senderDomain string) error

	// Start starts S2S router subsystem.
	Start(ctx context.Context) error

	// Stop stops S2S router subsystem.
	Stop(ctx context.Context) error
}

type router struct {
	hosts *host.Hosts
	c2s   C2SRouter
	s2s   S2SRouter
}

// New creates a new router instance given a set of hosts, C2S and S2s routers.
func New(hosts *host.Hosts, c2sRouter C2SRouter, s2sRouter S2SRouter) Router {
	return &router{
		hosts: hosts,
		c2s:   c2sRouter,
		s2s:   s2sRouter,
	}
}

func (r *router) Route(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
	return r.route(ctx, stanza, CheckUserExistence)
}

func (r *router) C2S() C2SRouter {
	return r.c2s
}

func (r *router) S2S() S2SRouter {
	return r.s2s
}

func (r *router) Start(ctx context.Context) error {
	if err := r.c2s.Start(ctx); err != nil {
		return err
	}
	if r.s2s == nil {
		return nil
	}
	return r.s2s.Start(ctx)
}

func (r *router) Stop(ctx context.Context) error {
	if err := r.c2s.Stop(ctx); err != nil {
		return err
	}
	if r.s2s == nil {
		return nil
	}
	return r.s2s.Stop(ctx)
}

func (r *router) route(ctx context.Context, stanza stravaganza.Stanza, routingOpts RoutingOptions) ([]jid.JID, error) {
	toJID := stanza.ToJID()
	if r.hosts.IsLocalHost(toJID.Domain()) {
		return r.c2s.Route(ctx, stanza, routingOpts)
	}
	if r.s2s == nil {
		return nil, ErrRemoteServerNotFound
	}
	if err := r.s2s.Route(ctx, stanza, r.hosts.DefaultHostName()); err != nil {
		return nil, err
	}
	return []jid.JID{*stanza.ToJID()}, nil
}
