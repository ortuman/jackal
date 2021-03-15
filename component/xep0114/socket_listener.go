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

package xep0114

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/component/extcomponentmanager"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/transport"
)

const (
	listenKeepAlive = time.Second * 15
)

// Options defines component connection options.
type Options struct {
	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration

	// KeepAliveTimeout defines the maximum amount of time that an inactive connection
	// would be considered alive.
	KeepAliveTimeout time.Duration

	// RequestTimeout defines component stream request timeout.
	RequestTimeout time.Duration

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int

	// Secret is the external component shared secret.
	Secret string
}

// SocketListener represents a component socket listener type.
type SocketListener struct {
	addr          string
	opts          Options
	hosts         *host.Hosts
	comps         *component.Components
	router        router.Router
	shapers       shaper.Shapers
	sn            *sonar.Sonar
	extCompMng    *extcomponentmanager.Manager
	stmHub        *inHub
	connHandlerFn func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewSocketListener returns a new external component socket listener.
func NewSocketListener(
	bindAddr string,
	port int,
	hosts *host.Hosts,
	comps *component.Components,
	extCompMng *extcomponentmanager.Manager,
	router router.Router,
	shapers shaper.Shapers,
	sn *sonar.Sonar,
	opts Options,
) *SocketListener {
	ln := &SocketListener{
		addr:       getAddress(bindAddr, port),
		hosts:      hosts,
		comps:      comps,
		router:     router,
		shapers:    shapers,
		sn:         sn,
		opts:       opts,
		stmHub:     newInHub(),
		extCompMng: extCompMng,
	}
	ln.connHandlerFn = ln.handleConn
	return ln
}

// Start starts listening on the TCP network address bindAddr to handle incoming connections.
func (l *SocketListener) Start(ctx context.Context) error {
	l.stmHub.start()

	lc := net.ListenConfig{
		KeepAlive: listenKeepAlive,
	}
	ln, err := lc.Listen(ctx, "tcp", l.addr)
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
			log.Infow(
				fmt.Sprintf("Received component incoming connection at %s", l.addr),
				"remote_address", conn.RemoteAddr().String(),
			)
			go l.connHandlerFn(conn)
		}
	}()
	log.Infof("Accepting external component connections at %s", l.addr)
	return nil
}

// Stop stops handling incoming connections and closes underlying TCP listener.
func (l *SocketListener) Stop(ctx context.Context) error {
	atomic.StoreUint32(&l.active, 0)
	if err := l.ln.Close(); err != nil {
		return err
	}
	l.stmHub.stop(ctx)

	log.Infof("Stopped external component listener at %s", l.addr)
	return nil
}

func (l *SocketListener) handleConn(conn net.Conn) {
	tr := transport.NewSocketTransport(conn)
	stm, err := newInComponent(
		tr,
		l.hosts,
		l.comps,
		l.extCompMng,
		l.stmHub,
		l.router,
		l.shapers,
		l.sn,
		l.opts,
	)
	if err != nil {
		log.Warnf("Failed to initialize component stream: %v", err)
		return
	}
	// start reading stream
	if err := stm.start(); err != nil {
		log.Warnf("Failed to start component stream: %v", err)
		return
	}
}

func getAddress(bindAddr string, port int) string {
	return bindAddr + ":" + strconv.Itoa(port)
}
