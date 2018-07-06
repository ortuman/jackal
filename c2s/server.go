/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // http profile handlers
	"strconv"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	cfg        *Config
	modConfig  *module.Config
	ln         net.Listener
	wsSrv      *http.Server
	wsUpgrader *websocket.Upgrader
	stmCounter uint64
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

	s.wsSrv = &http.Server{TLSConfig: &tls.Config{Certificates: host.Certificates()}}
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

func (s *server) shutdown() error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		switch s.cfg.Transport.Type {
		case transport.Socket:
			return s.ln.Close()
		case transport.WebSocket:
			return s.wsSrv.Close()
		}
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
		modules:          s.modConfig,
	}
	newStream(s.nextID(), cfg)
}

func (s *server) nextID() string {
	return fmt.Sprintf("c2s:%s:%d", s.cfg.ID, atomic.AddUint64(&s.stmCounter, 1))
}
