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
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/pkg/auth"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/transport"
)

const (
	listenKeepAlive = time.Second * 15

	scramSHA1Mechanism    = "scram_sha_1"
	scramSHA256Mechanism  = "scram_sha_256"
	scramSHA512Mechanism  = "scram_sha_512"
	scramSHA3512Mechanism = "scram_sha3_512"
)

// SocketListener represents a C2S socket listener type.
type SocketListener struct {
	addr           string
	cfg            Config
	saslMechanisms []string
	extAuth        *auth.External
	hosts          *host.Hosts
	router         router.Router
	comps          *component.Components
	mods           *module.Modules
	resMng         *ResourceManager
	rep            repository.Repository
	peppers        *pepper.Keys
	shapers        shaper.Shapers
	mh             *module.Hooks
	connHandlerFn  func(conn net.Conn)

	ln     net.Listener
	active uint32
}

// NewSocketListener returns a new C2S socket listener.
func NewSocketListener(
	bindAddr string,
	port int,
	saslMechanisms []string,
	extAuth *auth.External,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	resMng *ResourceManager,
	rep repository.Repository,
	peppers *pepper.Keys,
	shapers shaper.Shapers,
	mh *module.Hooks,
	cfg Config,
) *SocketListener {
	ln := &SocketListener{
		addr:           getAddress(bindAddr, port),
		saslMechanisms: saslMechanisms,
		extAuth:        extAuth,
		cfg:            cfg,
		hosts:          hosts,
		router:         router,
		comps:          comps,
		mods:           mods,
		resMng:         resMng,
		rep:            rep,
		peppers:        peppers,
		shapers:        shapers,
		mh:             mh,
	}
	ln.connHandlerFn = ln.handleConn
	return ln
}

// Start starts listening on the TCP network address bindAddr to handle incoming C2S connections.
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
	ln, err = lc.Listen(ctx, "tcp", l.addr)
	if err != nil {
		return err
	}
	if l.cfg.UseTLS {
		ln = tls.NewListener(ln, l.cfg.TLSConfig)
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
				fmt.Sprintf("Received C2S incoming connection at %s", l.addr),
				"remote_address", conn.RemoteAddr().String(),
			)

			go l.connHandlerFn(conn)
		}
	}()
	log.Infow(
		fmt.Sprintf("Accepting C2S socket connections at %s", l.addr),
		"direct_tls", l.cfg.UseTLS,
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
	log.Infof("Stopped C2S listener at %s", l.addr)
	return nil
}

func (l *SocketListener) handleConn(conn net.Conn) {
	tr := transport.NewSocketTransport(conn)
	stm, err := newInC2S(
		tr,
		l.getAuthenticators(tr),
		l.hosts,
		l.router,
		l.comps,
		l.mods,
		l.resMng,
		l.shapers,
		l.mh,
		l.cfg,
	)
	if err != nil {
		log.Warnf("Failed to initialize C2S stream: %v", err)
		return
	}
	// start reading stream
	if err := stm.start(); err != nil {
		log.Warnf("Failed to start C2S stream: %v", err)
		return
	}
}

func (l *SocketListener) getAuthenticators(tr transport.Transport) []auth.Authenticator {
	var res []auth.Authenticator
	if l.extAuth != nil {
		res = append(res, l.extAuth)
	}
	for _, mechanism := range l.saslMechanisms {
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
			log.Warnf("Unsupported authentication mechanism: %s", mechanism)
		}
	}
	return res
}

func getAddress(bindAddr string, port int) string {
	return bindAddr + ":" + strconv.Itoa(port)
}
