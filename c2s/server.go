/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/component"
	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	cfg        *Config
	mods       *module.Modules
	comps      *component.Components
	router     *router.Router
	inConns    sync.Map
	ln         net.Listener
	wsSrv      *http.Server
	wsUpgrader *websocket.Upgrader
	stmSeq     uint64
	listening  uint32
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
	case transport.WebSocket:
		err = s.listenWebSocketConn(address)
		break
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
			go s.startStream(transport.NewSocketTransport(conn, s.cfg.Transport.KeepAlive))
			continue
		}
	}
	return nil
}

func (s *server) listenWebSocketConn(address string) error {
	http.HandleFunc(s.cfg.Transport.URLPath, s.websocketUpgrade)

	s.wsSrv = &http.Server{TLSConfig: &tls.Config{Certificates: s.router.Certificates()}}
	s.wsUpgrader = &websocket.Upgrader{
		Subprotocols: []string{"xmpp"},
		CheckOrigin:  func(r *http.Request) bool { return r.Header.Get("Sec-WebSocket-Protocol") == "xmpp" },
	}

	// start listening
	ln, err := listenerProvider("tcp", address)
	if err != nil {
		return err
	}
	atomic.StoreUint32(&s.listening, 1)
	return s.wsSrv.ServeTLS(ln, "", "")
}

func (s *server) websocketUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	s.startStream(transport.NewWebSocketTransport(conn, s.cfg.Transport.KeepAlive))
}

func (s *server) shutdown(ctx context.Context) error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		// stop listening
		switch s.cfg.Transport.Type {
		case transport.Socket:
			if err := s.ln.Close(); err != nil {
				return err
			}
		case transport.WebSocket:
			if err := s.wsSrv.Shutdown(ctx); err != nil {
				return err
			}
		}
		// close all connections
		c, err := closeConnections(ctx, &s.inConns)
		if err != nil {
			return err
		}
		log.Infof("%s: closed %d connection(s)", s.cfg.ID, c)
	}
	return nil
}

func (s *server) startStream(tr transport.Transport) {
	cfg := &streamConfig{
		transport:        tr,
		resourceConflict: s.cfg.ResourceConflict,
		connectTimeout:   s.cfg.ConnectTimeout,
		maxStanzaSize:    s.cfg.MaxStanzaSize,
		sasl:             s.cfg.SASL,
		compression:      s.cfg.Compression,
		onDisconnect:     s.unregisterStream,
	}
	stm := newStream(s.nextID(), cfg, s.mods, s.comps, s.router)
	s.registerStream(stm)
}

func (s *server) registerStream(stm stream.C2S) {
	s.inConns.Store(stm.ID(), stm)
	log.Infof("registered c2s stream... (id: %s)", stm.ID())
}

func (s *server) unregisterStream(stm stream.C2S) {
	s.inConns.Delete(stm.ID())
	log.Infof("unregistered c2s stream... (id: %s)", stm.ID())
}

func (s *server) nextID() string {
	return fmt.Sprintf("c2s:%s:%d", s.cfg.ID, atomic.AddUint64(&s.stmSeq, 1))
}

func closeConnections(ctx context.Context, connections *sync.Map) (count int, err error) {
	connections.Range(func(_, v interface{}) bool {
		stm := v.(stream.InStream)
		select {
		case <-closeConn(stm):
			count++
			return true
		case <-ctx.Done():
			count = 0
			err = ctx.Err()
			return false
		}
	})
	return
}

func closeConn(stm stream.InStream) <-chan bool {
	c := make(chan bool, 1)
	go func() {
		stm.Disconnect(streamerror.ErrSystemShutdown)
		c <- true
	}()
	return c
}
