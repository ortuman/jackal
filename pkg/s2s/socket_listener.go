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

package s2s

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/transport"
)

const (
	listenKeepAlive = time.Second * 15
)

// SocketListener represents a S2S socket listener type.
type SocketListener struct {
	cfg           ListenerConfig
	hosts         *host.Hosts
	router        router.Router
	comps         *component.Components
	mods          *module.Modules
	outProvider   *OutProvider
	inHUB         *InHub
	kv            kv.KV
	shapers       shaper.Shapers
	hk            *hook.Hooks
	logger        kitlog.Logger
	connHandlerFn func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewListeners creates and initializes a set of S2S listeners based of cfg configuration.
func NewListeners(
	cfg ListenersConfig,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	outProvider *OutProvider,
	inHub *InHub,
	kv kv.KV,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	logger kitlog.Logger,
) []*SocketListener {
	var listeners []*SocketListener
	for _, lnCfg := range cfg {
		ln := newSocketListener(
			lnCfg,
			hosts,
			router,
			comps,
			mods,
			outProvider,
			kv,
			inHub,
			shapers,
			hk,
			logger,
		)
		listeners = append(listeners, ln)
	}
	return listeners
}

func newSocketListener(
	cfg ListenerConfig,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	outProvider *OutProvider,
	kv kv.KV,
	hub *InHub,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *SocketListener {
	ln := &SocketListener{
		cfg:         cfg,
		hosts:       hosts,
		router:      router,
		comps:       comps,
		mods:        mods,
		outProvider: outProvider,
		kv:          kv,
		inHUB:       hub,
		shapers:     shapers,
		hk:          hk,
		logger:      logger,
	}
	ln.connHandlerFn = ln.handleConn
	return ln
}

// Start starts listening on the TCP network address bindAddr to handle incoming S2S connections.
func (l *SocketListener) Start(ctx context.Context) error {
	var err error
	var ln net.Listener

	lc := net.ListenConfig{
		KeepAlive: listenKeepAlive,
	}
	ln, err = lc.Listen(ctx, "tcp", l.getAddress())
	if err != nil {
		return err
	}
	if l.cfg.DirectTLS {
		ln = tls.NewListener(ln, l.getTLSConfig())
	}
	l.ln = ln
	l.active = 1

	go func() {
		for atomic.LoadUint32(&l.active) == 1 {
			conn, err := l.ln.Accept()
			if err != nil {
				continue
			}
			level.Info(l.logger).Log("msg", "received S2S incoming connection",
				"bind_addr", l.getAddress(),
				"remote_address", conn.RemoteAddr().String(),
			)

			go l.connHandlerFn(conn)
		}
	}()
	level.Info(l.logger).Log("msg", "accepting S2S socket connections",
		"bind_addr", l.getAddress(),
		"direct_tls", l.cfg.DirectTLS,
	)
	return nil
}

// Stop stops handling incoming S2S connections and closes underlying TCP listener.
func (l *SocketListener) Stop(ctx context.Context) error {
	atomic.StoreUint32(&l.active, 0)
	if err := l.ln.Close(); err != nil {
		return err
	}
	level.Info(l.logger).Log("msg", "stopped S2S listener", "bind_addr", l.getAddress())
	return nil
}

func (l *SocketListener) handleConn(conn net.Conn) {
	tr := transport.NewSocketTransport(conn, l.cfg.ConnectTimeout, l.cfg.KeepAliveTimeout)
	stm, err := newInS2S(
		tr,
		l.hosts,
		l.router,
		l.comps,
		l.mods,
		l.outProvider,
		l.inHUB,
		l.kv,
		l.shapers,
		l.hk,
		l.logger,
		inConfig{
			reqTimeout:    l.cfg.RequestTimeout,
			maxStanzaSize: l.cfg.MaxStanzaSize,
			directTLS:     l.cfg.DirectTLS,
			tlsConfig:     l.getTLSConfig(),
		},
	)
	if err != nil {
		level.Warn(l.logger).Log("msg", "failed to initialize incoming S2S stream", "err", err)
		return
	}
	// start reading stream
	if err := stm.start(); err != nil {
		level.Warn(l.logger).Log("msg", "failed to start incoming S2S stream", "err", err)
		return
	}
}

func (l *SocketListener) getTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: l.hosts.Certificates(),
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}
}

func (l *SocketListener) getAddress() string {
	return l.cfg.BindAddr + ":" + strconv.Itoa(l.cfg.Port)
}
