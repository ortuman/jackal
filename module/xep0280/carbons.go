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

package xep0280

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	coremodel "github.com/ortuman/jackal/model/core"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
)

const (
	carbonsEnabledCtxKey = "carbons:enabled"

	carbonsNamespace          = "urn:xmpp:carbons:2"
	deliveryReceiptsNamespace = "urn:xmpp:receipts"
	forwardingNamespace       = "urn:xmpp:forward:0"
	chatStatesNamespace       = "http://jabber.org/protocol/chatstates"
	hintsNamespace            = "urn:xmpp:hints"
)

const (
	// ModuleName represents carbons module name.
	ModuleName = "carbons"

	// XEPNumber represents carbons XEP number.
	XEPNumber = "0280"
)

// Carbons represents carbons (XEP-0280) module type.
type Carbons struct {
	hosts  hosts
	router router.Router
	resMng resourceManager
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized carbons instance.
func New(hosts *host.Hosts, router router.Router, resMng *c2s.ResourceManager, sn *sonar.Sonar) *Carbons {
	return &Carbons{
		hosts:  hosts,
		router: router,
		resMng: resMng,
		sn:     sn,
	}
}

// Name returns carbons module name.
func (p *Carbons) Name() string { return ModuleName }

// StreamFeature returns carbons module stream feature.
func (p *Carbons) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns carbons server disco features.
func (p *Carbons) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{carbonsNamespace}, nil
}

// AccountFeatures returns ping account disco features.
func (p *Carbons) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts carbons module.
func (p *Carbons) Start(_ context.Context) error {
	p.subs = append(p.subs, p.sn.Subscribe(event.C2SStreamMessageRouted, p.onC2SMessageRouted))
	p.subs = append(p.subs, p.sn.Subscribe(event.S2SInStreamMessageRouted, p.onS2SMessageRouted))

	log.Infow("Started carbons module", "xep", XEPNumber)
	return nil
}

// Stop stops carbons module.
func (p *Carbons) Stop(_ context.Context) error {
	for _, sub := range p.subs {
		p.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped carbons module", "xep", XEPNumber)
	return nil
}

// MatchesNamespace tells whether namespace matches carbons module.
func (p *Carbons) MatchesNamespace(namespace string, serverTarget bool) bool {
	if serverTarget {
		return false
	}
	return namespace == carbonsNamespace
}

// ProcessIQ process a carbons iq.
func (p *Carbons) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsSet():
		return p.processIQ(ctx, iq)
	default:
		_, _ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
	}
	return nil
}

// Interceptors returns carbons stanza interceptor.
func (p *Carbons) Interceptors() []module.StanzaInterceptor {
	return []module.StanzaInterceptor{
		{Incoming: false},
	}
}

// InterceptStanza will be used by carbons module to strip private element before routing stanza.
func (p *Carbons) InterceptStanza(_ context.Context, stanza stravaganza.Stanza, _ int) (stravaganza.Stanza, error) {
	msg, ok := stanza.(*stravaganza.Message)
	if !ok {
		return stanza, nil
	}
	if msg.ChildNamespace("private", carbonsNamespace) == nil {
		return msg, nil
	}
	return stravaganza.NewBuilderFromElement(msg).
		WithoutChildrenNamespace("private", carbonsNamespace).
		BuildMessage()
}

func (p *Carbons) onC2SMessageRouted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)

	msg, ok := inf.Stanza.(*stravaganza.Message)
	if !ok {
		return nil
	}
	return p.processMessage(ctx, msg, inf.Targets)
}

func (p *Carbons) onS2SMessageRouted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.S2SStreamEventInfo)

	msg, ok := inf.Stanza.(*stravaganza.Message)
	if !ok {
		return nil
	}
	return p.processMessage(ctx, msg, nil)
}

func (p *Carbons) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	fromJID := iq.FromJID()
	if !p.hosts.IsLocalHost(fromJID.Domain()) {
		_, _ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.NotAllowed))
		return nil
	}
	switch {
	case iq.ChildNamespace("enable", carbonsNamespace) != nil:
		if err := p.setCarbonsEnabled(ctx, fromJID.Node(), fromJID.Resource(), true); err != nil {
			return err
		}
		log.Infow("Enabled carbons copy", "username", fromJID.Node(), "resource", fromJID.Resource())

	case iq.ChildNamespace("disable", carbonsNamespace) != nil:
		if err := p.setCarbonsEnabled(ctx, fromJID.Node(), fromJID.Resource(), false); err != nil {
			return err
		}
		log.Infow("Disabled carbons copy", "username", fromJID.Node(), "resource", fromJID.Resource())

	default:
		_, _ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}

	_, _ = p.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
	return nil
}

func (p *Carbons) setCarbonsEnabled(ctx context.Context, username, resource string, enabled bool) error {
	stm := p.router.C2S().LocalStream(username, resource)
	if stm == nil {
		return errStreamNotFound(username, resource)
	}
	return stm.SetValue(ctx, carbonsEnabledCtxKey, strconv.FormatBool(enabled))
}

func (p *Carbons) processMessage(ctx context.Context, msg *stravaganza.Message, ignoringTargets []jid.JID) error {
	if !isEligibleMessage(msg) || isPrivateMessage(msg) || isCCMessage(msg) {
		return nil
	}
	fromJID := msg.FromJID()
	toJID := msg.ToJID()

	if fromJID.IsFullWithUser() && p.hosts.IsLocalHost(fromJID.Domain()) {
		if err := p.routeSentCC(ctx, msg, fromJID.Node()); err != nil {
			return err
		}
	}
	if !toJID.IsServer() && p.hosts.IsLocalHost(toJID.Domain()) {
		if err := p.routeReceivedCC(ctx, msg, toJID.Node(), ignoringTargets); err != nil {
			return err
		}
	}
	return nil
}

func (p *Carbons) routeSentCC(ctx context.Context, msg *stravaganza.Message, username string) error {
	rss, err := p.getFilteredResources(ctx, username, []jid.JID{*msg.FromJID()})
	if err != nil {
		return err
	}
	for _, res := range rss {
		enabled, _ := strconv.ParseBool(res.Value(carbonsEnabledCtxKey))
		if !enabled {
			continue
		}
		_, _ = p.router.Route(ctx, sentMsgCC(msg, res.JID))
	}
	return nil
}

func (p *Carbons) routeReceivedCC(ctx context.Context, msg *stravaganza.Message, username string, ignoringTargets []jid.JID) error {
	rss, err := p.getFilteredResources(ctx, username, ignoringTargets)
	if err != nil {
		return err
	}
	for _, res := range rss {
		enabled, _ := strconv.ParseBool(res.Value(carbonsEnabledCtxKey))
		if !enabled {
			continue
		}
		_, _ = p.router.Route(ctx, receivedMsgCC(msg, res.JID))
	}
	return nil
}

func (p *Carbons) getFilteredResources(ctx context.Context, username string, ignoringJIDs []jid.JID) ([]coremodel.Resource, error) {
	rs, err := p.resMng.GetResources(ctx, username)
	if err != nil {
		return nil, err
	}
	ignoredJIDs := make(map[string]struct{}, len(ignoringJIDs))
	for _, j := range ignoringJIDs {
		ignoredJIDs[j.String()] = struct{}{}
	}
	var ret []coremodel.Resource
	for _, res := range rs {
		_, ok := ignoredJIDs[res.JID.String()]
		if ok {
			continue
		}
		ret = append(ret, res)
	}
	return ret, nil
}

func isEligibleMessage(msg *stravaganza.Message) bool {
	if msg.Attribute(stravaganza.Type) == stravaganza.ChatType {
		return true
	}
	if msg.Attribute(stravaganza.Type) == stravaganza.NormalType && msg.IsMessageWithBody() {
		return true
	}
	for _, ch := range msg.AllChildren() {
		cns := ch.Attribute(stravaganza.Namespace)
		if cns == deliveryReceiptsNamespace || cns == chatStatesNamespace {
			return true
		}
	}
	return false
}

func isPrivateMessage(msg *stravaganza.Message) bool {
	return msg.ChildNamespace("private", carbonsNamespace) != nil && msg.ChildNamespace("no-copy", hintsNamespace) != nil
}

func isCCMessage(msg *stravaganza.Message) bool {
	return msg.ChildNamespace("sent", carbonsNamespace) != nil || msg.ChildNamespace("received", carbonsNamespace) != nil
}

func sentMsgCC(msg *stravaganza.Message, dest *jid.JID) *stravaganza.Message {
	ccMsg, _ := stravaganza.NewMessageBuilder().
		WithAttribute(stravaganza.From, dest.ToBareJID().String()).
		WithAttribute(stravaganza.To, dest.String()).
		WithAttribute(stravaganza.Type, stravaganza.ChatType).
		WithChild(
			stravaganza.NewBuilder("sent").
				WithAttribute(stravaganza.Namespace, carbonsNamespace).
				WithChild(
					stravaganza.NewBuilder("forwarded").
						WithAttribute(stravaganza.Namespace, forwardingNamespace).
						WithChild(msg).
						Build(),
				).
				Build(),
		).
		BuildMessage()
	return ccMsg
}

func receivedMsgCC(msg *stravaganza.Message, dest *jid.JID) *stravaganza.Message {
	ccMsg, _ := stravaganza.NewMessageBuilder().
		WithAttribute(stravaganza.From, dest.ToBareJID().String()).
		WithAttribute(stravaganza.To, dest.String()).
		WithAttribute(stravaganza.Type, stravaganza.ChatType).
		WithChild(
			stravaganza.NewBuilder("received").
				WithAttribute(stravaganza.Namespace, carbonsNamespace).
				WithChild(
					stravaganza.NewBuilder("forwarded").
						WithAttribute(stravaganza.Namespace, forwardingNamespace).
						WithChild(msg).
						Build(),
				).
				Build(),
		).
		BuildMessage()
	return ccMsg
}

func errStreamNotFound(username, resource string) error {
	return fmt.Errorf("xep0280: local stream not found: %s/%s", username, resource)
}
