/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/pkg/errors"
)

const (
	streamNamespace           = "http://etherx.jabber.org/streams"
	tlsNamespace              = "urn:ietf:params:xml:ns:xmpp-tls"
	compressProtocolNamespace = "http://jabber.org/protocol/compress"
	bindNamespace             = "urn:ietf:params:xml:ns:xmpp-bind"
	sessionNamespace          = "urn:ietf:params:xml:ns:xmpp-session"
	saslNamespace             = "urn:ietf:params:xml:ns:xmpp-sasl"
	blockedErrorNamespace     = "urn:xmpp:blocking:errors"
)

type c2sServer interface {
	start()
	shutdown(ctx context.Context) error
}

var createC2SServer = newC2SServer

// C2S represents a client-to-server connection manager.
type C2S struct {
	mu      sync.RWMutex
	servers map[string]c2sServer
	started uint32
}

// New returns a new instance of a c2s connection manager.
func New(configs []Config, mods *module.Modules, comps *component.Components, router router.Router, userRep repository.User, blockListRep repository.BlockList) (*C2S, error) {
	if len(configs) == 0 {
		return nil, errors.New("at least one c2s configuration is required")
	}
	c := &C2S{servers: make(map[string]c2sServer)}
	for _, config := range configs {
		srv := createC2SServer(&config, mods, comps, router, userRep, blockListRep)
		c.servers[config.ID] = srv
	}
	return c, nil
}

// Start initializes c2s manager spawning every single server.
func (c *C2S) Start() {
	if atomic.CompareAndSwapUint32(&c.started, 0, 1) {
		for _, srv := range c.servers {
			go srv.start()
		}
	}
}

// Shutdown gracefully shuts down c2s manager.
func (c *C2S) Shutdown(ctx context.Context) {
	if atomic.CompareAndSwapUint32(&c.started, 1, 0) {
		for _, srv := range c.servers {
			if err := srv.shutdown(ctx); err != nil {
				log.Error(err)
			}
		}
	}
}
