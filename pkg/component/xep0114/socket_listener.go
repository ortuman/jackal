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

	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/component/extcomponentmanager"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/transport"
)

const (
	listenKeepAlive = time.Second * 15
)

// SocketListener represents a component socket listener type.
type SocketListener struct {
	cfg           ListenerConfig
	hosts         *host.Hosts
	comps         *component.Components
	router        router.Router
	shapers       shaper.Shapers
	hk            *hook.Hooks
	extCompMng    *extcomponentmanager.Manager
	stmHub        *inHub
	connHandlerFn func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewListeners creates and initializes a set of component listeners based of cfg configuration.
func NewListeners(
	cfg ListenersConfig,
	hosts *host.Hosts,
	comps *component.Components,
	extCompMng *extcomponentmanager.Manager,
	router router.Router,
	shapers shaper.Shapers,
	hk *hook.Hooks,
) []*SocketListener {
	var listeners []*SocketListener
	for _, lnCfg := range cfg {
		ln := newSocketListener(
			lnCfg,
			hosts,
			comps,
			extCompMng,
			router,
			shapers,
			hk,
		)
		listeners = append(listeners, ln)
	}
	return listeners
}

func newSocketListener(
	cfg ListenerConfig,
	hosts *host.Hosts,
	comps *component.Components,
	extCompMng *extcomponentmanager.Manager,
	router router.Router,
	shapers shaper.Shapers,
	hk *hook.Hooks,
) *SocketListener {
	ln := &SocketListener{
		hosts:      hosts,
		comps:      comps,
		router:     router,
		shapers:    shapers,
		hk:         hk,
		cfg:        cfg,
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
	ln, err := lc.Listen(ctx, "tcp", l.getAddress())
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
				fmt.Sprintf("Received component incoming connection at %s", l.getAddress()),
				"remote_address", conn.RemoteAddr().String(),
			)
			go l.connHandlerFn(conn)
		}
	}()
	log.Infof("accepting external component connections at %s", l.getAddress())
	return nil
}

// Stop stops handling incoming connections and closes underlying TCP listener.
func (l *SocketListener) Stop(ctx context.Context) error {
	atomic.StoreUint32(&l.active, 0)
	if err := l.ln.Close(); err != nil {
		return err
	}
	l.stmHub.stop(ctx)

	log.Infof("stopped external component listener at %s", l.getAddress())
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
		l.hk,
		inConfig{
			connectTimeout:   l.cfg.ConnectTimeout,
			keepAliveTimeout: l.cfg.KeepAliveTimeout,
			reqTimeout:       l.cfg.RequestTimeout,
			maxStanzaSize:    l.cfg.MaxStanzaSize,
			secret:           l.cfg.Secret,
		},
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

func (l *SocketListener) getAddress() string {
	return l.cfg.BindAddr + ":" + strconv.Itoa(l.cfg.Port)
}
