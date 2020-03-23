/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/component"
	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	cfg             *Config
	mods            *module.Modules
	comps           *component.Components
	router          router.Router
	userRep         repository.User
	blockListRep    repository.BlockList
	inConnectionsMu sync.Mutex
	inConnections   map[string]stream.C2S
	ln              net.Listener
	stmSeq          uint64
	listening       uint32
}

func newC2SServer(config *Config, mods *module.Modules, comps *component.Components, router router.Router, userRep repository.User, blockListRep repository.BlockList) c2sServer {
	return &server{
		cfg:           config,
		mods:          mods,
		comps:         comps,
		router:        router,
		userRep:       userRep,
		blockListRep:  blockListRep,
		inConnections: make(map[string]stream.C2S),
	}
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %v]", s.cfg.ID, address, s.cfg.Transport.Type)

	var err error
	switch s.cfg.Transport.Type {
	case transport.Socket:
		err = s.listenSocketConn(address)
	}
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *server) listenSocketConn(address string) error {
	ln, err := listenerProvider("tcp", address)
	if err != nil {
		return err
	}
	s.ln = ln

	atomic.StoreUint32(&s.listening, 1)
	for atomic.LoadUint32(&s.listening) == 1 {
		conn, err := ln.Accept()
		if err == nil {
			go s.startStream(transport.NewSocketTransport(conn), s.cfg.KeepAlive)
			continue
		}
	}
	return nil
}

func (s *server) shutdown(ctx context.Context) error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		// stop listening
		switch s.cfg.Transport.Type {
		case transport.Socket:
			if err := s.ln.Close(); err != nil {
				return err
			}
		}
		// close all connections
		c, err := s.closeConnections(ctx)
		if err != nil {
			return err
		}
		log.Infof("%s: closed %d connection(s)", s.cfg.ID, c)
	}
	return nil
}

func (s *server) startStream(tr transport.Transport, keepAlive time.Duration) {
	cfg := &streamConfig{
		resourceConflict: s.cfg.ResourceConflict,
		connectTimeout:   s.cfg.ConnectTimeout,
		keepAlive:        s.cfg.KeepAlive,
		timeout:          s.cfg.Timeout,
		maxStanzaSize:    s.cfg.MaxStanzaSize,
		sasl:             s.cfg.SASL,
		compression:      s.cfg.Compression,
		onDisconnect:     s.unregisterStream,
	}
	stm := newStream(s.nextID(), cfg, tr, s.mods, s.comps, s.router, s.userRep, s.blockListRep)
	s.registerStream(stm)
}

func (s *server) registerStream(stm stream.C2S) {
	s.inConnectionsMu.Lock()
	s.inConnections[stm.ID()] = stm
	s.inConnectionsMu.Unlock()

	log.Infof("registered c2s stream... (id: %s)", stm.ID())
}

func (s *server) unregisterStream(stm stream.C2S) {
	s.inConnectionsMu.Lock()
	delete(s.inConnections, stm.ID())
	s.inConnectionsMu.Unlock()

	log.Infof("unregistered c2s stream... (id: %s)", stm.ID())
}

func (s *server) nextID() string {
	return fmt.Sprintf("c2s:%s:%d", s.cfg.ID, atomic.AddUint64(&s.stmSeq, 1))
}

func (s *server) closeConnections(ctx context.Context) (count int, err error) {
	s.inConnectionsMu.Lock()
	for _, stm := range s.inConnections {
		select {
		case <-closeConn(ctx, stm):
			count++
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	s.inConnectionsMu.Unlock()
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
