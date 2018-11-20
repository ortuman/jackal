/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
)

const streamMailboxSize = 256

const (
	streamNamespace   = "http://etherx.jabber.org/streams"
	tlsNamespace      = "urn:ietf:params:xml:ns:xmpp-tls"
	saslNamespace     = "urn:ietf:params:xml:ns:xmpp-sasl"
	dialbackNamespace = "urn:xmpp:features:dialback"
)

// S2S represents a server-to-server connection manager.
type S2S struct {
	srv     *server
	started uint32
}

// New returns a new instance of an s2s connection manager.
func New(config *Config, mods *module.Modules, router *router.Router) *S2S {
	if config == nil {
		return nil
	}
	return &S2S{
		srv: &server{cfg: config, router: router, mods: mods, dialer: newDialer(config, router)},
	}
}

// GetS2SOut acts as an s2s outgoing stream provider.
func (s *S2S) GetS2SOut(localDomain, remoteDomain string) (stream.S2SOut, error) {
	if s.srv == nil {
		return nil, errors.New("s2s not initialized")
	}
	return s.srv.getOrDial(localDomain, remoteDomain)
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
