/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"errors"
	"sync"

	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/stream"
)

const streamMailboxSize = 256

const (
	streamNamespace   = "http://etherx.jabber.org/streams"
	tlsNamespace      = "urn:ietf:params:xml:ns:xmpp-tls"
	saslNamespace     = "urn:ietf:params:xml:ns:xmpp-sasl"
	dialbackNamespace = "urn:xmpp:features:dialback"
)

var (
	instMu        sync.RWMutex
	defaultDialer *dialer
	srv           *server
	initialized   bool
)

func Initialize(cfg *Config, modConfig *module.Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	if !cfg.Enabled {
		return
	}
	defaultDialer = newDialer(cfg)
	srv = &server{cfg: cfg, modConfig: modConfig}
	go srv.start()
	initialized = true
}

func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		srv.shutdown()
		srv = nil
		initialized = false
	}
}

func GetS2SOut(localDomain, remoteDomain string) (stream.S2SOut, error) {
	instMu.RLock()
	if !initialized {
		instMu.RUnlock()
		return nil, errors.New("s2s not available")
	}
	instMu.RUnlock()
	return outContainer.getOrDial(localDomain, remoteDomain)
}
