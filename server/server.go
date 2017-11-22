/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package server

import (
	"time"

	"github.com/ortuman/jackal/config"
)

const (
	// C2S represents a client to client server type.
	C2S = iota
	// S2S represents a server-to-client server type.
	S2S
)

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
	time.Sleep(time.Second * 5)
}
