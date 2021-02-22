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

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/ortuman/jackal/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type httpServer struct {
	port int
	srv  *http.Server
}

func newHTTPServer(port int) *httpServer {
	return &httpServer{port: port}
}

func (h *httpServer) Start(_ context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))

	h.srv = &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
	if err != nil {
		return err
	}
	go func() {
		if err := h.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Errorf("Failed to serve HTTP: %v", err)
		}
	}()
	log.Infof("HTTP server listening at :%d", h.port)
	return nil
}

func (h *httpServer) Stop(ctx context.Context) error {
	if err := h.srv.Shutdown(ctx); err != nil {
		return err
	}
	log.Infof("Closed HTTP server at :%d", h.port)
	return nil
}
