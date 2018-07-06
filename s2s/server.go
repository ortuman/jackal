/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"net"
	"strconv"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	cfg       *Config
	modConfig *module.Config
	ln        net.Listener
	listening uint32
}

func (s *server) start() {
	bindAddr := s.cfg.Transport.BindAddress
	port := s.cfg.Transport.Port
	address := bindAddr + ":" + strconv.Itoa(port)

	log.Infof("s2s_in: listening at %s", address)

	if err := s.listenConn(address); err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *server) shutdown() {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		s.ln.Close()
	}
}

func (s *server) listenConn(address string) error {
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

func (s *server) startStream(tr transport.Transport) {
	newInStream(&streamConfig{
		modConfig:      s.modConfig,
		keyGen:         &keyGen{s.cfg.DialbackSecret},
		transport:      tr,
		connectTimeout: s.cfg.ConnectTimeout,
		maxStanzaSize:  s.cfg.MaxStanzaSize,
		dialer:         newDialerCopy(defaultDialer),
	})
}
