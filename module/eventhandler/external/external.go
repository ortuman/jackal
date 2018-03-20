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

package eventhandlerexternal

import (
	"context"
	"fmt"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/cluster/instance"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	eventhandlerpb "github.com/ortuman/jackal/module/eventhandler/external/pb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

var dialExtConnFn = dialExtConn

// Handler represents an external event handler.
type Handler struct {
	address     string
	isSecure    bool
	topics      []string
	sonar       *sonar.Sonar
	cc          *grpc.ClientConn
	cl          eventhandlerpb.EventHandlerClient
	subs        []sonar.SubID
	srvFeatures []string
	accFeatures []string
}

// New returns a new initialized Handler instance.
func New(address string, isSecure bool, topics []string, sonar *sonar.Sonar) *Handler {
	return &Handler{
		address:  address,
		isSecure: isSecure,
		topics:   topics,
		sonar:    sonar,
	}
}

// Name returns module name.
func (h *Handler) Name() string { return "ext-eventhandler" }

// ServerFeatures returns module server features.
func (h *Handler) ServerFeatures() []string {
	return h.srvFeatures
}

// AccountFeatures returns module account features.
func (h *Handler) AccountFeatures() []string {
	return h.accFeatures
}

// Start starts external event handler.
func (h *Handler) Start(ctx context.Context) error {
	// dial external event handler conn
	cl, cc, err := dialExtConnFn(ctx, h.address, h.isSecure)
	if err != nil {
		return err
	}
	h.cc = cc
	h.cl = cl

	// get module disco features
	fResp, err := h.cl.GetDiscoFeatures(ctx, &eventhandlerpb.GetDiscoFeaturesRequest{})
	if err != nil {
		return err
	}
	h.srvFeatures = fResp.GetServerFeatures()
	h.accFeatures = fResp.GetAccountFeatures()

	// subscribe to handler events
	for _, topic := range h.topics {
		h.subs = append(h.subs, h.sonar.Subscribe(topic, h.onEvent))
	}
	log.Infow(fmt.Sprintf("Configured external event handler at %s", h.address), "secured", h.isSecure, "topics", h.topics)
	return nil
}

// Stop stops external event handler.
func (h *Handler) Stop(_ context.Context) error {
	// unsubscribe from handler events
	for _, sub := range h.subs {
		h.sonar.Unsubscribe(sub)
	}
	return h.cc.Close()
}

func (h *Handler) onEvent(ctx context.Context, ev sonar.Event) error {
	_, err := h.cl.ProcessEvent(ctx, toPBProcessEventRequest(ev.Name(), ev.Info()))
	if err != nil {
		return errors.Wrap(err, "eventhandler: failed to process event")
	}
	return nil
}

func toPBProcessEventRequest(evName string, evInfo interface{}) *eventhandlerpb.ProcessEventRequest {
	ret := &eventhandlerpb.ProcessEventRequest{
		InstanceId: instance.ID(),
		EventName:  evName,
	}
	switch inf := evInfo.(type) {
	case *event.C2SStreamEventInfo:
		var evInf eventhandlerpb.C2SStreamEventInfo
		evInf.Id = inf.ID
		if inf.JID != nil {
			evInf.Jid = inf.JID.String()
		}
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &eventhandlerpb.ProcessEventRequest_C2SStreamEvInfo{
			C2SStreamEvInfo: &evInf,
		}

	case *event.S2SStreamEventInfo:
		var evInf eventhandlerpb.S2SStreamEventInfo
		evInf.Id = inf.ID
		evInf.Sender = inf.Sender
		evInf.Target = inf.Target
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &eventhandlerpb.ProcessEventRequest_S2SStreamEvInfo{
			S2SStreamEvInfo: &evInf,
		}

	case *event.ExternalComponentEventInfo:
		var evInf eventhandlerpb.ExternalComponentEventInfo
		evInf.Id = inf.ID
		evInf.Host = inf.Host
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &eventhandlerpb.ProcessEventRequest_ExtComponentEvInfo{
			ExtComponentEvInfo: &evInf,
		}

	case *event.RosterEventInfo:
		ret.Payload = &eventhandlerpb.ProcessEventRequest_RosterEvInfo{
			RosterEvInfo: &eventhandlerpb.RosterEventInfo{
				Username:     inf.Username,
				Jid:          inf.JID,
				Subscription: inf.Subscription,
			},
		}

	case *event.VCardEventInfo:
		ret.Payload = &eventhandlerpb.ProcessEventRequest_VcardEvInfo{
			VcardEvInfo: &eventhandlerpb.VCardEventInfo{
				Username: inf.Username,
			},
		}

	case *event.OfflineEventInfo:
		ret.Payload = &eventhandlerpb.ProcessEventRequest_OfflineEvInfo{
			OfflineEvInfo: &eventhandlerpb.OfflineEventInfo{
				Username: inf.Username,
				Message:  inf.Message.Proto(),
			},
		}

	case *event.UserEventInfo:
		ret.Payload = &eventhandlerpb.ProcessEventRequest_UserEvInfo{
			UserEvInfo: &eventhandlerpb.UserEventInfo{
				Username: inf.Username,
			},
		}
	}
	return ret
}

func dialExtConn(ctx context.Context, addr string, isSecure bool) (eventhandlerpb.EventHandlerClient, *grpc.ClientConn, error) {
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
	return eventhandlerpb.NewEventHandlerClient(cc), cc, nil
}
