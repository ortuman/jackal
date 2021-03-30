// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clusterserver

import (
	"context"
	"net"
	"strconv"
	"sync/atomic"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/ortuman/jackal/c2s"
	clusterpb "github.com/ortuman/jackal/cluster/pb"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/log"
	"google.golang.org/grpc"
)

var netListen = net.Listen

// Server represents cluster server type.
type Server struct {
	bindAddr    string
	port        int
	ln          net.Listener
	srv         *grpc.Server
	active      int32
	localRouter *c2s.LocalRouter
	comps       *component.Components
}

// New returns a new initialized Server instance.
func New(bindAddr string, port int, localRouter *c2s.LocalRouter, comps *component.Components) *Server {
	return &Server{
		bindAddr:    bindAddr,
		port:        port,
		localRouter: localRouter,
		comps:       comps,
	}
}

// Start starts cluster server.
func (s *Server) Start(_ context.Context) error {
	addr := s.getAddress()

	ln, err := netListen("tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	s.active = 1

	log.Infof("Started cluster server at %s", addr)

	s.srv = grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	clusterpb.RegisterLocalRouterServer(s.srv, newLocalRouterService(s.localRouter))
	clusterpb.RegisterComponentRouterServer(s.srv, newComponentRouterService(s.comps))

	go func() {
		if err := s.srv.Serve(s.ln); err != nil {
			if atomic.LoadInt32(&s.active) == 1 {
				log.Errorf("Cluster server error: %s", err)
			}
		}
	}()
	return nil
}

// Stop stops cluster server.
func (s *Server) Stop(_ context.Context) error {
	atomic.StoreInt32(&s.active, 0)
	s.srv.GracefulStop()

	log.Infof("Closed cluster server at %s", s.getAddress())
	return nil
}

func (s *Server) getAddress() string {
	return s.bindAddr + ":" + strconv.Itoa(s.port)
}
