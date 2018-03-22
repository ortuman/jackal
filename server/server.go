/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // http profile handlers
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/stream/c2s"
)

type server struct {
	ln         net.Listener
	wsSrv      *http.Server
	wsUpgrader *websocket.Upgrader
	cfg        *config.Server
	strCounter int32
	listening  uint32
}

var (
	servers     = map[string]*server{}
	shutdownCh  = make(chan bool)
	debugSrv    *http.Server
	initialized uint32
)

// Initialize spawns a connection listener for every server configuration.
func Initialize(srvConfigurations []config.Server, debugPort int) {
	if !atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		return
	}
	if debugPort > 0 {
		// initialize debug service
		go func() {
			debugSrv = &http.Server{Addr: fmt.Sprintf(":%d", debugPort)}
			debugSrv.ListenAndServe()
		}()
	}

	// initialize all servers
	for i := 0; i < len(srvConfigurations); i++ {
		initializeServer(&srvConfigurations[i])
	}

	// wait until shutdown...
	<-shutdownCh

	// close all servers
	for k, srv := range servers {
		if err := srv.shutdown(); err != nil {
			log.Error(err)
		}
		delete(servers, k)
	}
}

// Shutdown closes every server listener.
// This method should be used only for testing purposes.
func Shutdown() {
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		if debugSrv != nil {
			debugSrv.Close()
		}
		shutdownCh <- true
	}
}

func initializeServer(srvConfig *config.Server) {
	srv := &server{cfg: srvConfig}
	servers[srvConfig.ID] = srv
	go srv.start()
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %v]", s.cfg.ID, address, s.cfg.Transport.Type)

	switch s.cfg.Transport.Type {
	case config.SocketTransportType:
		s.listenSocketConn(address)
	case config.WebSocketTransportType:
		s.listenWebSocketConn(address)
		break
	}
}

func (s *server) listenSocketConn(address string) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	s.ln = ln

	atomic.StoreUint32(&s.listening, 1)
	for atomic.LoadUint32(&s.listening) == 1 {
		conn, err := ln.Accept()
		if err == nil {
			go s.handleSocketConn(conn)
			continue
		}
	}
}

func (s *server) listenWebSocketConn(address string) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("%v", err)
		}
	}()

	cer, err := tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.PrivKeyFile)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	cfg := &tls.Config{
		ServerName:   c2s.Instance().DefaultLocalDomain(),
		Certificates: []tls.Certificate{cer},
	}
	wsSrv := &http.Server{
		Addr:      address,
		TLSConfig: cfg,
	}
	s.wsUpgrader = &websocket.Upgrader{
		ReadBufferSize:  s.cfg.Transport.BufferSize,
		WriteBufferSize: s.cfg.Transport.BufferSize,
		Subprotocols:    []string{"xmpp"},
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Sec-WebSocket-Protocol") == "xmpp"
		},
	}
	s.wsSrv = wsSrv

	http.HandleFunc(fmt.Sprintf("/%s/ws", url.PathEscape(s.cfg.ID)), s.websocketUpgrade)

	atomic.StoreUint32(&s.listening, 1)
	if err := s.wsSrv.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *server) websocketUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	go s.handleWebSocketConn(conn)
}

func (s *server) shutdown() error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		switch s.cfg.Transport.Type {
		case config.SocketTransportType:
			return s.ln.Close()
		case config.WebSocketTransportType:
			return s.wsSrv.Close()
		}
	}
	return nil
}

func (s *server) handleSocketConn(conn net.Conn) {
	s.startStream(transport.NewSocketTransport(conn, s.cfg.Transport.BufferSize, s.cfg.Transport.KeepAlive))
}

func (s *server) handleWebSocketConn(conn *websocket.Conn) {
	s.startStream(transport.NewWebSocketTransport(conn, s.cfg.Transport.KeepAlive))
}

func (s *server) startStream(tr transport.Transport) {
	stm := newStream(s.nextID(), tr, s.cfg)
	if err := c2s.Instance().RegisterStream(stm); err != nil {
		log.Error(err)
	}
}

func (s *server) nextID() string {
	return fmt.Sprintf("%s:%d", s.cfg.ID, atomic.AddInt32(&s.strCounter, 1))
}
