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

package c2s

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	"github.com/ortuman/jackal/pkg/auth"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/ortuman/jackal/pkg/transport/compress"
)

const (
	listenKeepAlive = time.Second * 15

	scramSHA1Mechanism    = "scram_sha_1"
	scramSHA256Mechanism  = "scram_sha_256"
	scramSHA512Mechanism  = "scram_sha_512"
	scramSHA3512Mechanism = "scram_sha3_512"
)

var cmpLevelMap = map[string]compress.Level{
	"default": compress.DefaultCompression,
	"best":    compress.BestCompression,
	"speed":   compress.SpeedCompression,
}

var resConflictMap = map[string]resourceConflict{
	"override":      override,
	"disallow":      disallow,
	"terminate_old": terminateOld,
}

// SocketListener represents a C2S socket listener type.
type SocketListener struct {
	cfg     ListenerConfig
	extAuth *auth.External
	hosts   *host.Hosts
	router  router.Router
	comps   *component.Components
	mods    *module.Modules
	resMng  *ResourceManager
	rep     repository.Repository
	peppers *pepper.Keys
	shapers shaper.Shapers
	hk      *hook.Hooks
	logger  kitlog.Logger

	tlsCfg        *tls.Config
	connHandlerFn func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewListeners creates and initializes a set of C2S listeners based of cfg configuration.
func NewListeners(
	cfg ListenersConfig,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	resMng *ResourceManager,
	rep repository.Repository,
	peppers *pepper.Keys,
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
			resMng,
			rep,
			peppers,
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
	resMng *ResourceManager,
	rep repository.Repository,
	peppers *pepper.Keys,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *SocketListener {
	var extAuth *auth.External
	if len(cfg.SASL.External.Address) > 0 {
		extAuth = auth.NewExternal(
			cfg.SASL.External.Address,
			cfg.SASL.External.IsSecure,
		)
	}
	ln := &SocketListener{
		cfg:     cfg,
		extAuth: extAuth,
		hosts:   hosts,
		router:  router,
		comps:   comps,
		mods:    mods,
		resMng:  resMng,
		rep:     rep,
		peppers: peppers,
		shapers: shapers,
		hk:      hk,
		logger:  logger,
	}
	ln.connHandlerFn = ln.handleConn
	return ln
}

// Start starts listening on a TCP network address to handle incoming C2S connections.
func (l *SocketListener) Start(ctx context.Context) error {
	if l.extAuth != nil {
		// dial external authenticator
		if err := l.extAuth.Start(ctx); err != nil {
			return err
		}
	}
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
		l.tlsCfg = &tls.Config{
			Certificates: l.hosts.Certificates(),
			MinVersion:   tls.VersionTLS12,
		}
		ln = tls.NewListener(ln, l.tlsCfg)
	}
	l.ln = ln
	l.active = 1

	go func() {
		for atomic.LoadUint32(&l.active) == 1 {
			conn, err := l.ln.Accept()
			if err != nil {
				continue
			}
			level.Info(l.logger).Log("msg", "received C2S incoming connection",
				"bind_addr", l.getAddress(),
				"remote_address", conn.RemoteAddr().String(),
			)

			go l.connHandlerFn(conn)
		}
	}()
	level.Info(l.logger).Log("msg", "accepting C2S socket connections",
		"bind_addr", l.getAddress(),
		"direct_tls", l.cfg.DirectTLS,
	)
	return nil
}

// Stop stops handling incoming C2S connections and closes underlying TCP listener.
func (l *SocketListener) Stop(ctx context.Context) error {
	atomic.StoreUint32(&l.active, 0)
	if err := l.ln.Close(); err != nil {
		return err
	}
	if l.extAuth != nil {
		// close external authenticator conn
		if err := l.extAuth.Stop(ctx); err != nil {
			return err
		}
	}
	level.Info(l.logger).Log("msg", "stopped C2S listener", "bind_addr", l.getAddress())
	return nil
}

func (l *SocketListener) handleConn(conn net.Conn) {
	tr := transport.NewSocketTransport(conn, l.cfg.ConnectTimeout, l.cfg.KeepAliveTimeout)
	stm, err := newInC2S(
		l.getInConfig(),
		tr,
		l.getAuthenticators(tr),
		l.hosts,
		l.router,
		l.comps,
		l.mods,
		l.resMng,
		l.shapers,
		l.hk,
		l.logger,
	)
	if err != nil {
		level.Warn(l.logger).Log("msg", "failed to initialize C2S stream", "err", err)
		return
	}
	// start reading stream
	if err := stm.start(); err != nil {
		level.Warn(l.logger).Log("msg", "failed to start C2S stream", "err", err)
		return
	}
}

func (l *SocketListener) getAuthenticators(tr transport.Transport) []auth.Authenticator {
	var res []auth.Authenticator
	if l.extAuth != nil {
		res = append(res, l.extAuth)
	}
	for _, mechanism := range l.cfg.SASL.Mechanisms {
		switch mechanism {
		case scramSHA1Mechanism:
			res = append(res, auth.NewScram(tr, auth.ScramSHA1, false, l.rep, l.peppers))
			res = append(res, auth.NewScram(tr, auth.ScramSHA1, true, l.rep, l.peppers))

		case scramSHA256Mechanism:
			res = append(res, auth.NewScram(tr, auth.ScramSHA256, false, l.rep, l.peppers))
			res = append(res, auth.NewScram(tr, auth.ScramSHA256, true, l.rep, l.peppers))

		case scramSHA512Mechanism:
			res = append(res, auth.NewScram(tr, auth.ScramSHA512, false, l.rep, l.peppers))
			res = append(res, auth.NewScram(tr, auth.ScramSHA512, true, l.rep, l.peppers))

		case scramSHA3512Mechanism:
			res = append(res, auth.NewScram(tr, auth.ScramSHA3512, false, l.rep, l.peppers))
			res = append(res, auth.NewScram(tr, auth.ScramSHA3512, true, l.rep, l.peppers))
		default:
			level.Warn(l.logger).Log("msg", "unsupported authentication mechanism", "mechanism", mechanism)
		}
	}
	return res
}

func (l *SocketListener) getInConfig() inCfg {
	return inCfg{
		authenticateTimeout: l.cfg.AuthenticateTimeout,
		reqTimeout:          l.cfg.RequestTimeout,
		maxStanzaSize:       l.cfg.MaxStanzaSize,
		compressionLevel:    cmpLevelMap[l.cfg.CompressionLevel],
		resConflict:         resConflictMap[l.cfg.ResourceConflict],
		useTLS:              l.cfg.DirectTLS,
		tlsConfig:           l.tlsCfg,
	}
}

func (l *SocketListener) getAddress() string {
	return l.cfg.BindAddr + ":" + strconv.Itoa(l.cfg.Port)
}
