/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package server

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/ortuman/jackal/stream"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
)

const defaultServerPort = 5222

const defaultMaxStanzaSize = 65536
const defaultConnectTimeout = 5
const defaultKeepAlive = 120

type server struct {
	cfg         config.Server
	strmCounter int32
}

func Initialize() {
	for i := 1; i < len(config.DefaultConfig.Servers); i++ {
		go initializeServer(config.DefaultConfig.Servers[i])
	}
	initializeServer(config.DefaultConfig.Servers[0])
}

func initializeServer(serverConfig config.Server) {
	srv := newServerWithConfig(serverConfig)
	srv.start()
}

func newServerWithConfig(serverConfig config.Server) *server {
	s := &server{
		cfg: serverConfig,
	}
	return s
}

func (s *server) start() {
	// validate server's configuration
	if err := s.validateConfiguration(); err != nil {
		log.Errorf("%v", err)
		return
	}

	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	if port == 0 {
		port = defaultServerPort
	}
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("%s: listening at %s [transport: %s]", s.cfg.ID, address, s.cfg.Transport.Type)

	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%v", err)
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
	strm := stream.NewStreamSocket(id, conn, &s.cfg)
	stream.Manager().RegisterStream(strm)
}

func (s *server) validateConfiguration() error {
	// validate server type
	switch s.cfg.Type {
	case config.C2S, config.S2S:
		break
	default:
		return fmt.Errorf("unrecognized server type: %s", s.cfg.Type)
	}
	// validate domain
	if len(s.cfg.Domains) == 0 {
		return errors.New("no domain specified")
	}

	// validate transport
	if s.cfg.Transport.Type != config.SocketTransport {
		return fmt.Errorf("unrecognized transport type: %s", s.cfg.Transport.Type)
	}

	// assign transport default values
	if s.cfg.Transport.Port == 0 {
		s.cfg.Transport.Port = defaultServerPort
	}
	if s.cfg.Transport.MaxStanzaSize == 0 {
		s.cfg.Transport.MaxStanzaSize = defaultMaxStanzaSize
	}
	if s.cfg.Transport.ConnectTimeout == 0 {
		s.cfg.Transport.ConnectTimeout = defaultConnectTimeout
	}
	if s.cfg.Transport.KeepAlive == 0 {
		s.cfg.Transport.KeepAlive = defaultKeepAlive
	}
	return nil
}
