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
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/util"
)

type server struct {
	cfg        *Config
	tlsCfg     *tls.Config
	ln         net.Listener
	wsSrv      *http.Server
	wsUpgrader *websocket.Upgrader
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
func Initialize(srvConfigurations []Config, debugPort int) {
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
		if err := initializeServer(&srvConfigurations[i]); err != nil {
			log.Fatalf("%v", err)
		}
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

func newServer(cfg *Config) (*server, error) {
	s := &server{cfg: cfg}
	tlsCfg, err := util.LoadCertificate(s.cfg.TLS.PrivKeyFile, s.cfg.TLS.CertFile, router.Instance().DefaultLocalDomain())
	if err != nil {
		return nil, err
	}
	s.tlsCfg = tlsCfg
	return s, nil
}

func initializeServer(cfg *Config) error {
	srv, err := newServer(cfg)
	if err != nil {
		return err
	}
	servers[cfg.ID] = srv
	go srv.start()
	return nil
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %v]", s.cfg.ID, address, s.cfg.Transport.Type)

	switch s.cfg.Transport.Type {
	case transport.Socket:
		s.listenSocketConn(address)
	case transport.WebSocket:
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
			go s.startStream(transport.NewSocketTransport(conn, s.cfg.Transport.KeepAlive))
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
	wsSrv := &http.Server{
		Addr:      address,
		TLSConfig: s.tlsCfg,
	}
	s.wsUpgrader = &websocket.Upgrader{
		Subprotocols: []string{"xmpp"},
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
	go s.startStream(transport.NewWebSocketTransport(conn, s.cfg.Transport.KeepAlive))
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
	stm := c2s.New(s.nextID(), tr, s.tlsCfg, &s.cfg.C2S)
	if err := router.Instance().RegisterStream(stm); err != nil {
		log.Error(err)
	}
}

func (s *server) nextID() string {
	return fmt.Sprintf("%s:%d", s.cfg.ID, atomic.AddInt32(&s.strCounter, 1))
}
