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

package iqhandlerexternal

import (
	"context"
	"fmt"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/log"
	iqhandlerpb "github.com/ortuman/jackal/module/iqhandler/external/pb"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/util/stringmatcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

var dialExtConnFn = dialExtConn

// Handler represents an external iq handler.
type Handler struct {
	address     string
	isSecure    bool
	router      router.Router
	nsMatcher   stringmatcher.Matcher
	cc          *grpc.ClientConn
	cl          iqhandlerpb.IQHandlerClient
	srvFeatures []string
	accFeatures []string
}

// New returns a new initialized Handler instance.
func New(address string, isSecure bool, namespaceMatcher stringmatcher.Matcher, router router.Router) *Handler {
	h := &Handler{
		address:   address,
		isSecure:  isSecure,
		router:    router,
		nsMatcher: namespaceMatcher,
	}
	return h
}

// Name returns module name.
func (h *Handler) Name() string { return "ext-iqhandler" }

// StreamFeature returns module stream feature.
func (h *Handler) StreamFeature() stravaganza.Element { return nil }

// ServerFeatures returns module server features.
func (h *Handler) ServerFeatures() []string {
	return h.srvFeatures
}

// AccountFeatures returns module account features.
func (h *Handler) AccountFeatures() []string {
	return h.accFeatures
}

// MatchesNamespace tells whether namespace matches extern iq handler.
func (h *Handler) MatchesNamespace(namespace string) bool {
	return h.nsMatcher.Matches(namespace)
}

// ProcessIQ will be invoked whenever iq stanza should be processed by this module.
func (h *Handler) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	resp, err := h.cl.ProcessIQ(ctx, &iqhandlerpb.ProcessIQRequest{
		Iq: iq.Proto(),
	})
	if err != nil {
		return err
	}
	for _, pbStanza := range resp.GetRespStanzas() {
		stanza, err := stravaganza.NewBuilderFromProto(pbStanza).BuildStanza(true)
		if err != nil {
			log.Warnf("Failed to process external IQ handler response stanza: %v", err)
			continue
		}
		_ = h.router.Route(ctx, stanza)
	}
	return nil
}

// Start starts external iq handler.
func (h *Handler) Start(ctx context.Context) error {
	// dial external IQ handler conn
	cl, cc, err := dialExtConnFn(ctx, h.address, h.isSecure)
	if err != nil {
		return err
	}
	h.cc = cc
	h.cl = cl

	// get module disco features
	fResp, err := h.cl.GetDiscoFeatures(ctx, &iqhandlerpb.GetDiscoFeaturesRequest{})
	if err != nil {
		return err
	}
	h.srvFeatures = fResp.GetServerFeatures()
	h.accFeatures = fResp.GetAccountFeatures()

	log.Infow(fmt.Sprintf("Configured external IQ handler at %s", h.address), "secured", h.isSecure)
	return nil
}

// Stop stops external iq handler.
func (h *Handler) Stop(_ context.Context) error {
	return h.cc.Close()
}

func dialExtConn(ctx context.Context, addr string, isSecure bool) (iqhandlerpb.IQHandlerClient, *grpc.ClientConn, error) {
	var opts = []grpc.DialOption{
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if !isSecure {
		opts = append(opts, grpc.WithInsecure())
	}
	cc, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, nil, err
	}
	return iqhandlerpb.NewIQHandlerClient(cc), cc, nil
}
