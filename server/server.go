/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"

	// pprof
	_ "net/http/pprof"

	"github.com/ortuman/jackal/stream"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
)

type server struct {
	cfg         *config.Server
	strmCounter int32
}

func Initialize() {
	// initialize debug
	if config.DefaultConfig.Debug != nil {
		go func() {
			http.ListenAndServe(fmt.Sprintf(":%d", config.DefaultConfig.Debug.Port), nil)
		}()
	}

	for i := 1; i < len(config.DefaultConfig.Servers); i++ {
		go initializeServer(&config.DefaultConfig.Servers[i])
	}
	initializeServer(&config.DefaultConfig.Servers[0])
}

func initializeServer(serverConfig *config.Server) {
	srv := newServerWithConfig(serverConfig)
	srv.start()
}

func newServerWithConfig(serverConfig *config.Server) *server {
	s := &server{
		cfg: serverConfig,
	}
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
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("%v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *server) handleConnection(conn net.Conn) {
	id := fmt.Sprintf("%s:%d", s.cfg.ID, atomic.AddInt32(&s.strmCounter, 1))
	strm := stream.NewStreamSocket(id, conn, s.cfg)
	stream.Manager().RegisterStream(strm)
}
