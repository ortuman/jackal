// Copyright 2020 The jackal Authors
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

package s2s

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/cluster/kv"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/transport"
)

var (
	listen    = net.Listen
	listenTLS = tls.Listen
)

// SocketListener represents a S2S socket listener type.
type SocketListener struct {
	addr          string
	hosts         *host.Hosts
	router        router.Router
	comps         *component.Components
	mods          *module.Modules
	outProvider   *OutProvider
	inHub         *InHub
	kv            kv.KV
	shapers       shaper.Shapers
	sonar         *sonar.Sonar
	opts          Options
	connHandlerFn func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewSocketListener returns a new S2S socket listener.
func NewSocketListener(
	bindAddr string,
	port int,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	outProvider *OutProvider,
	inHub *InHub,
	kv kv.KV,
	shapers shaper.Shapers,
	sonar *sonar.Sonar,
	opts Options,
) *SocketListener {
	addr := getAddress(bindAddr, port)
	ln := &SocketListener{
		addr:        addr,
		opts:        opts,
		hosts:       hosts,
		router:      router,
		comps:       comps,
		mods:        mods,
		outProvider: outProvider,
		inHub:       inHub,
		kv:          kv,
		shapers:     shapers,
		sonar:       sonar,
	}
	ln.connHandlerFn = ln.handleConn
	return ln
}

// Start starts listening on the TCP network address bindAddr to handle incoming S2S connections.
func (l *SocketListener) Start(_ context.Context) error {
	var err error
	var ln net.Listener

	if l.opts.UseTLS {
		ln, err = listenTLS("tcp", l.addr, l.opts.TLSConfig)
	} else {
		ln, err = listen("tcp", l.addr)
	}
	if err != nil {
		return err
	}
	l.ln = ln
	l.active = 1

	go func() {
		for atomic.LoadUint32(&l.active) == 1 {
			conn, err := l.ln.Accept()
			if err != nil {
				continue
			}
			go l.connHandlerFn(conn)
		}
	}()
	log.Infow(fmt.Sprintf("Accepting S2S socket connections at %s", l.addr),
		"direct_tls", l.opts.UseTLS,
	)
	return nil
}

// Stop stops handling incoming S2S connections and closes underlying TCP listener.
func (l *SocketListener) Stop(_ context.Context) error {
	atomic.StoreUint32(&l.active, 0)
	if err := l.ln.Close(); err != nil {
		return err
	}
	log.Infof("Stopped S2S listener at %s", l.addr)
	return nil
}

func (l *SocketListener) handleConn(conn net.Conn) {
	tr := transport.NewSocketTransport(conn)
	stm, err := newInS2S(
		tr,
		l.hosts,
		l.router,
		l.comps,
		l.mods,
		l.outProvider,
		l.inHub,
		l.kv,
		l.shapers,
		l.sonar,
		l.opts,
	)
	if err != nil {
		log.Warnf("Failed to initialize incoming S2S stream: %v", err)
		return
	}
	// start reading stream
	if err := stm.start(); err != nil {
		log.Warnf("Failed to start incoming S2S stream: %v", err)
		return
	}
}

func getAddress(bindAddr string, port int) string {
	return bindAddr + ":" + strconv.Itoa(port)
}
