/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
)

const (
	streamNamespace   = "http://etherx.jabber.org/streams"
	tlsNamespace      = "urn:ietf:params:xml:ns:xmpp-tls"
	saslNamespace     = "urn:ietf:params:xml:ns:xmpp-sasl"
	dialbackNamespace = "urn:xmpp:features:dialback"
)

type s2sServer interface {
	start()
	shutdown(ctx context.Context) error
}

var createS2SServer = func(config *Config, mods *module.Modules, newOutFn newOutFunc, router router.Router) s2sServer {
	return newServer(
		config,
		mods,
		newOutFn,
		router,
	)
}

// S2S represents a server-to-server connection manager.
type S2S struct {
	started     uint32
	srv         s2sServer
	outProvider *OutProvider
}

// New returns a new instance of an s2s connection manager.
func New(config *Config, mods *module.Modules, outProvider *OutProvider, router router.Router) *S2S {
	return &S2S{srv: createS2SServer(config, mods, outProvider.newOut, router)}
}

// Start initializes s2s manager.
func (s *S2S) Start() {
	if atomic.CompareAndSwapUint32(&s.started, 0, 1) {
		go s.srv.start()
	}
}

// Shutdown gracefully shuts down s2s manager.
func (s *S2S) Shutdown(ctx context.Context) {
	if atomic.CompareAndSwapUint32(&s.started, 1, 0) {
		if err := s.srv.shutdown(ctx); err != nil {
			log.Error(err)
		}
	}
}
