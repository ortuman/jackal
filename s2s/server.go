/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
)

var listenerProvider = net.Listen

type server struct {
	cfg       *Config
	router    *router.Router
	mods      *module.Modules
	dialer    *dialer
	inConns   sync.Map
	outConns  sync.Map
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

func (s *server) shutdown(ctx context.Context) error {
	if atomic.CompareAndSwapUint32(&s.listening, 1, 0) {
		// stop listening...
		if err := s.ln.Close(); err != nil {
			return err
		}
		// close all connections...
		c, err := closeConnections(&s.outConns, ctx)
		if err != nil {
			return err
		}
		log.Infof("%s: closed %d out connection(s)", s.cfg.ID, c)

		c, err = closeConnections(&s.inConns, ctx)
		if err != nil {
			return err
		}
		log.Infof("%s: closed %d in connection(s)", s.cfg.ID, c)
	}
	return nil
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
			go s.startInStream(transport.NewSocketTransport(conn, s.cfg.Transport.KeepAlive))
			continue
		}
	}
	return nil
}

func (s *server) getOrDial(localDomain, remoteDomain string) (stream.S2SOut, error) {
	domainPair := localDomain + ":" + remoteDomain
	stm, loaded := s.outConns.LoadOrStore(domainPair, newOutStream(s.router))
	if !loaded {
		outCfg, err := s.dialer.dial(localDomain, remoteDomain)
		if err != nil {
			log.Error(err)
			s.outConns.Delete(domainPair)
			return nil, err
		}
		outCfg.onOutDisconnect = s.unregisterOutStream

		stm.(*outStream).start(outCfg)
		log.Infof("registered s2s out stream... (domainpair: %s)", domainPair)
	}
	return stm.(*outStream), nil
}

func (s *server) unregisterOutStream(stm stream.S2SOut) {
	domainPair := stm.ID()
	s.outConns.Delete(domainPair)
	log.Infof("unregistered s2s out stream... (domainpair: %s)", domainPair)
}

func (s *server) startInStream(tr transport.Transport) {
	stm := newInStream(&streamConfig{
		keyGen:         &keyGen{s.cfg.DialbackSecret},
		transport:      tr,
		connectTimeout: s.cfg.ConnectTimeout,
		maxStanzaSize:  s.cfg.MaxStanzaSize,
		dialer:         s.dialer,
		onInDisconnect: s.unregisterInStream,
	}, s.mods, s.router)
	s.registerInStream(stm)
}

func (s *server) registerInStream(stm stream.S2SIn) {
	s.inConns.Store(stm.ID(), stm)
	log.Infof("registered s2s in stream... (id: %s)", stm.ID())
}

func (s *server) unregisterInStream(stm stream.S2SIn) {
	s.inConns.Delete(stm.ID())
	log.Infof("unregistered s2s in stream... (id: %s)", stm.ID())
}

func closeConnections(connections *sync.Map, ctx context.Context) (count int, err error) {
	connections.Range(func(_, v interface{}) bool {
		stm := v.(*inStream)
		select {
		case <-closeConn(stm):
			count++
			return true
		case <-ctx.Done():
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
