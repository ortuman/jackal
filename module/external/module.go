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

package externalmodule

import (
	"context"
	"fmt"
	"io"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/cluster/instance"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	extmodulepb "github.com/ortuman/jackal/module/external/pb"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/util/stringmatcher"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

const (
	serverTargetEntity  = "server"
	accountTargetEntity = "account"
)

// Config defines external module configuratiopn value.
type Config struct {
	// RequestTimeout defines external module request timeout.
	RequestTimeout time.Duration

	// Topics defines all topics to which a external module wants to subscribe to.
	Topics []string

	// NamespaceMatcher defines external module IQ namespace matcher.
	NamespaceMatcher stringmatcher.Matcher

	// TargetEntity specifies target entity type for which module IQ handler should be applied.
	TargetEntity string

	// Interceptors contains external module StanzaInterceptor set.
	Interceptors []module.StanzaInterceptor
}

var dialExtConnFn = dialExtConn

// ExtModule represents an external module.
type ExtModule struct {
	name     string
	address  string
	isSecure bool
	cfg      Config
	sonar    *sonar.Sonar
	subs     []sonar.SubID
	router   router.Router

	cc io.Closer
	cl extmodulepb.ModuleClient
}

// New returns a new initialized ExtModule instance.
func New(
	address string,
	isSecure bool,
	router router.Router,
	sonar *sonar.Sonar,
	cfg Config,
) *ExtModule {
	return &ExtModule{
		address:  address,
		isSecure: isSecure,
		cfg:      cfg,
		router:   router,
		sonar:    sonar,
	}
}

// Name returns external module name.
func (m *ExtModule) Name() string {
	return m.name
}

// StreamFeature returns external module stream feature elements.
func (m *ExtModule) StreamFeature(ctx context.Context, domain string) (stravaganza.Element, error) {
	resp, err := m.cl.GetStreamFeature(ctx, &extmodulepb.GetStreamFeatureRequest{
		Domain: domain,
	})
	if err != nil {
		return nil, err
	}
	return stravaganza.NewBuilderFromProto(resp.GetFeature()).Build(), nil
}

// ServerFeatures returns module server features.
func (m *ExtModule) ServerFeatures(ctx context.Context) ([]string, error) {
	resp, err := m.cl.GetServerFeatures(ctx, &extmodulepb.GetServerFeaturesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetFeatures(), nil
}

// AccountFeatures returns module account features.
func (m *ExtModule) AccountFeatures(ctx context.Context) ([]string, error) {
	resp, err := m.cl.GetAccountFeatures(ctx, &extmodulepb.GetAccountFeaturesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetFeatures(), nil
}

// MatchesNamespace tells whether namespace matches external iq handler.
func (m *ExtModule) MatchesNamespace(namespace string, serverTarget bool) bool {
	if m.cfg.NamespaceMatcher == nil {
		return false
	}
	switch {
	case serverTarget:
		if m.cfg.TargetEntity == accountTargetEntity {
			return false
		}
	default:
		if m.cfg.TargetEntity == serverTargetEntity {
			return false
		}
	}
	return m.cfg.NamespaceMatcher.Matches(namespace)
}

// ProcessIQ will be invoked whenever iq stanza should be processed by this external module.
func (m *ExtModule) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	_, err := m.cl.ProcessIQ(ctx, &extmodulepb.ProcessIQRequest{
		Iq: iq.Proto(),
	})
	return err
}

// Interceptors returns a set of all module interceptors.
func (m *ExtModule) Interceptors() []module.StanzaInterceptor {
	return m.cfg.Interceptors
}

// InterceptStanza will be invoked to allow stanza transformation based on a StanzaInterceptor definition.
func (m *ExtModule) InterceptStanza(ctx context.Context, stanza stravaganza.Stanza, id int) (stravaganza.Stanza, error) {
	resp, err := m.cl.InterceptStanza(ctx, &extmodulepb.InterceptStanzaRequest{
		Id:     int64(id),
		Stanza: stanza.Proto(),
	})
	if err != nil {
		return nil, err
	}
	if resp.Interrupt {
		return nil, module.ErrInterceptStanzaInterrupted
	}
	// ensure stanza type
	switch stanza.(type) {
	case *stravaganza.IQ:
		return stravaganza.NewBuilderFromProto(resp.Stanza).
			BuildIQ()
	case *stravaganza.Presence:
		return stravaganza.NewBuilderFromProto(resp.Stanza).
			BuildPresence()
	case *stravaganza.Message:
		return stravaganza.NewBuilderFromProto(resp.Stanza).
			BuildMessage()
	default:
		return stravaganza.NewBuilderFromProto(resp.Stanza).
			BuildStanza()
	}
}

// Start starts external module.
func (m *ExtModule) Start(ctx context.Context) error {
	// dial external module conn
	cl, cc, err := dialExtConnFn(ctx, m.address, m.isSecure)
	if err != nil {
		return err
	}
	m.cc = cc
	m.cl = cl

	// start reading incoming stanzas
	stm, err := m.cl.GetStanzas(ctx, &extmodulepb.GetStanzasRequest{})
	if err != nil {
		return err
	}
	go m.recvStanzas(stm)

	// subscribe to handler events
	for _, topic := range m.cfg.Topics {
		m.subs = append(m.subs, m.sonar.Subscribe(topic, m.onEvent))
	}
	log.Infow(fmt.Sprintf("Started %s external module at: %s", m.name, m.address),
		"secured", m.isSecure,
	)
	return nil
}

// Stop stops external module.
func (m *ExtModule) Stop(_ context.Context) error {
	// unsubscribe from external module events
	for _, sub := range m.subs {
		m.sonar.Unsubscribe(sub)
	}
	if err := m.cc.Close(); err != nil {
		return err
	}
	log.Infow(fmt.Sprintf("Stopped %s external module", m.name))
	return nil
}

func (m *ExtModule) onEvent(ctx context.Context, ev sonar.Event) error {
	_, err := m.cl.ProcessEvent(ctx, toPBProcessEventRequest(ev.Name(), ev.Info()))
	if err != nil {
		return errors.Wrap(err, "externalmodule: failed to process event")
	}
	return nil
}

func (m *ExtModule) recvStanzas(stm extmodulepb.Module_GetStanzasClient) {
	for {
		proto, err := stm.Recv()
		switch err {
		case nil:
			stanza, err := stravaganza.NewBuilderFromProto(proto).
				WithValidateJIDs(true).
				BuildStanza()
			if err != nil {
				log.Warnf("externalmodule: failed to process incoming stanza: %v", err)
				continue
			}
			ctx, cancel := m.requestContext()
			_, _ = m.router.Route(ctx, stanza)
			cancel()

		case io.EOF:
			return

		default:
			log.Warnf("externalmodule: failed to receive stanza: %v", err)
		}
	}
}

func (m *ExtModule) requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), m.cfg.RequestTimeout)
}

func toPBProcessEventRequest(evName string, evInfo interface{}) *extmodulepb.ProcessEventRequest {
	ret := &extmodulepb.ProcessEventRequest{
		InstanceId: instance.ID(),
		EventName:  evName,
	}
	switch inf := evInfo.(type) {
	case *event.C2SStreamEventInfo:
		var evInf extmodulepb.C2SStreamEventInfo
		evInf.Id = inf.ID
		if inf.JID != nil {
			evInf.Jid = inf.JID.String()
		}
		for _, target := range inf.Targets {
			evInf.Targets = append(evInf.Targets, target.String())
		}
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &extmodulepb.ProcessEventRequest_C2SStreamEvInfo{
			C2SStreamEvInfo: &evInf,
		}

	case *event.S2SStreamEventInfo:
		var evInf extmodulepb.S2SStreamEventInfo
		evInf.Id = inf.ID
		evInf.Sender = inf.Sender
		evInf.Target = inf.Target
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &extmodulepb.ProcessEventRequest_S2SStreamEvInfo{
			S2SStreamEvInfo: &evInf,
		}

	case *event.ExternalComponentEventInfo:
		var evInf extmodulepb.ExternalComponentEventInfo
		evInf.Id = inf.ID
		evInf.Host = inf.Host
		if inf.Stanza != nil {
			evInf.Stanza = inf.Stanza.Proto()
		}
		ret.Payload = &extmodulepb.ProcessEventRequest_ExtComponentEvInfo{
			ExtComponentEvInfo: &evInf,
		}

	case *event.RosterEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_RosterEvInfo{
			RosterEvInfo: &extmodulepb.RosterEventInfo{
				Username:     inf.Username,
				Jid:          inf.JID,
				Subscription: inf.Subscription,
			},
		}

	case *event.PrivateEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_PrivateEvInfo{
			PrivateEvInfo: &extmodulepb.PrivateEventInfo{
				Username: inf.Username,
				Private:  inf.Private.Proto(),
			},
		}

	case *event.VCardEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_VcardEvInfo{
			VcardEvInfo: &extmodulepb.VCardEventInfo{
				Username: inf.Username,
				Vcard:    inf.VCard.Proto(),
			},
		}

	case *event.OfflineEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_OfflineEvInfo{
			OfflineEvInfo: &extmodulepb.OfflineEventInfo{
				Username: inf.Username,
				Message:  inf.Message.Proto(),
			},
		}

	case *event.UserEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_UserEvInfo{
			UserEvInfo: &extmodulepb.UserEventInfo{
				Username: inf.Username,
			},
		}

	case *event.BlockListEventInfo:
		var evInf extmodulepb.BlockListEventInfo
		evInf.Username = inf.Username
		for _, jd := range inf.JIDs {
			evInf.Jids = append(evInf.Jids, jd.String())
		}
		ret.Payload = &extmodulepb.ProcessEventRequest_BlocklistEvInfo{
			BlocklistEvInfo: &evInf,
		}

	case *event.LastActivityEventInfo:
		ret.Payload = &extmodulepb.ProcessEventRequest_LastEvInfo{
			LastEvInfo: &extmodulepb.LastEventInfo{
				Username: inf.Username,
				Jid:      inf.JID.String(),
			},
		}
	}
	return ret
}

func dialExtConn(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
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
	return extmodulepb.NewModuleClient(cc), cc, nil
}
