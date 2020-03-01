/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	mu            sync.RWMutex
	cfg           *Config
	router        router.Router
	mods          *module.Modules
	newOutFn      newOutFunc
	inConnections map[string]stream.S2SIn
	ln            net.Listener
	listening     uint32
}

func newServer(config *Config, mods *module.Modules, newOutFn newOutFunc, router router.Router) *server {
	return &server{
		cfg:           config,
		router:        router,
		mods:          mods,
		newOutFn:      newOutFn,
		inConnections: make(map[string]stream.S2SIn),
	}
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("s2s_in: listening at %s", address)

	if err := s.listenConn(address); err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *server) shutdown(ctx context.Context) error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		// stop listening...
		if err := s.ln.Close(); err != nil {
			return err
		}
		// close all connections...
		c, err := s.closeConnections(ctx)
		if err != nil {
			return err
		}
		log.Infof("%s: closed %d in connection(s)", s.cfg.ID, c)
	}
	return nil
}

func (s *server) listenConn(address string) error {
	ln, err := listenerProvider("tcp", address)
	if err != nil {
		return err
	}
	s.ln = ln

	atomic.StoreUint32(&s.listening, 1)
	for atomic.LoadUint32(&s.listening) == 1 {
		conn, err := ln.Accept()
		if err == nil {
			go s.startInStream(transport.NewSocketTransport(conn, s.cfg.Transport.KeepAlive))
			continue
		}
	}
	return nil
}

func (s *server) startInStream(tr transport.Transport) {
	stm := newInStream(&inConfig{
		keyGen:         &keyGen{s.cfg.DialbackSecret},
		transport:      tr,
		connectTimeout: s.cfg.ConnectTimeout,
		timeout:        s.cfg.Timeout,
		maxStanzaSize:  s.cfg.MaxStanzaSize,
		onDisconnect:   s.unregisterInStream,
	}, s.mods, s.newOutFn, s.router)
	s.registerInStream(stm)
}

func (s *server) registerInStream(stm stream.S2SIn) {
	s.mu.Lock()
	s.inConnections[stm.ID()] = stm
	s.mu.Unlock()

	log.Infof("registered s2s in stream... (id: %s)", stm.ID())
}

func (s *server) unregisterInStream(stm stream.S2SIn) {
	s.mu.Lock()
	delete(s.inConnections, stm.ID())
	s.mu.Unlock()

	log.Infof("unregistered s2s in stream... (id: %s)", stm.ID())
}

func (s *server) closeConnections(ctx context.Context) (count int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, stm := range s.inConnections {
		select {
		case <-closeConn(ctx, stm):
			count++
		case <-ctx.Done():
			err = ctx.Err()
			return 0, err
		}
	}
	return count, nil
}

func closeConn(ctx context.Context, stm stream.InStream) <-chan bool {
	c := make(chan bool, 1)
	go func() {
		stm.Disconnect(ctx, streamerror.ErrSystemShutdown)
		c <- true
	}()
	return c
}
