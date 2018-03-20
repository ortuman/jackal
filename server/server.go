/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // http profile handlers
	"strconv"
	"sync/atomic"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/c2s"
)

type server struct {
	ln         net.Listener
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
	for _, srvConfig := range srvConfigurations {
		initializeServer(&srvConfig)
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

func initializeServer(serverConfig *config.Server) {
	srv := newServerWithConfig(serverConfig)
	servers[serverConfig.ID] = srv
	go srv.start()
}

func newServerWithConfig(serverConfig *config.Server) *server {
	s := &server{cfg: serverConfig}
	return s
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %v]", s.cfg.ID, address, s.cfg.Transport.Type)

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
			go s.handleConnection(conn)
			continue
		}
	}
}

func (s *server) shutdown() error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		return s.ln.Close()
	}
	return nil
}

func (s *server) handleConnection(conn net.Conn) {
	id := fmt.Sprintf("%s:%d", s.cfg.ID, atomic.AddInt32(&s.strCounter, 1))
	stm := newSocketStream(id, conn, s.cfg)
	if err := c2s.Instance().RegisterStream(stm); err != nil {
		log.Error(err)
	}
}
