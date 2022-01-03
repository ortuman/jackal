// Copyright 2021 The jackal Authors
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

package jackal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type httpServer struct {
	port   int
	srv    *http.Server
	logger kitlog.Logger
}

func newHTTPServer(port int, logger kitlog.Logger) *httpServer {
	return &httpServer{port: port, logger: logger}
}

func (h *httpServer) Start(_ context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	h.srv = &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
	if err != nil {
		return err
	}
	go func() {
		if err := h.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			level.Error(h.logger).Log("msg", "failed to serve HTTP", "err", err)
		}
	}()
	level.Info(h.logger).Log("msg", "HTTP server listening", "port", h.port)
	return nil
}

func (h *httpServer) Stop(ctx context.Context) error {
	if err := h.srv.Shutdown(ctx); err != nil {
		return err
	}
	level.Info(h.logger).Log("msg", "closed HTTP server", "port", h.port)
	return nil
}
