/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package server

import (
	"net"
	"strconv"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
)

const (
	// C2S represents a client to client server type.
	C2S = iota
	// S2S represents a server-to-client server type.
	S2S
)

const defaultServerPort = 5222

type server struct {
	cfg *config.Server
}

func Initialize() {
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
	if port == 0 {
		port = defaultServerPort
	}
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %s]", s.cfg.ID, address, s.cfg.Transport.Type)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("%v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *server) handleConnection(conn net.Conn) {
	println(conn)
}
