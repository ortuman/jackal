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
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/util"
)

var listenerProvider = net.Listen

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
		go startDebugServer(debugPort)
	}

	// initialize all servers
	for i := 0; i < len(srvConfigurations); i++ {
		if _, err := initializeServer(&srvConfigurations[i]); err != nil {
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

func initializeServer(cfg *Config) (*server, error) {
	srv, err := newServer(cfg)
	if err != nil {
		return nil, err
	}
	servers[cfg.ID] = srv
	go srv.start()
	return srv, nil
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

func startDebugServer(port int) {
	debugSrv = &http.Server{}
	ln, err := listenerProvider("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("%v", err)
	}
	debugSrv.Serve(ln)
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
			keepAlive := time.Second * time.Duration(s.cfg.Transport.KeepAlive)
			go s.startStream(transport.NewSocketTransport(conn, keepAlive))
			continue
		}
	}
	return nil
}

func (s *server) listenWebSocketConn(address string) error {
	http.HandleFunc(fmt.Sprintf("/%s/ws", url.PathEscape(s.cfg.ID)), s.websocketUpgrade)

	s.wsSrv = &http.Server{TLSConfig: s.tlsCfg}
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
	stm := c2s.New(s.nextID(), tr, s.tlsCfg, &s.cfg.C2S)
	if err := router.Instance().RegisterC2S(stm); err != nil {
		log.Error(err)
	}
}

func (s *server) nextID() string {
	return fmt.Sprintf("%s:%d", s.cfg.ID, atomic.AddInt32(&s.strCounter, 1))
}
