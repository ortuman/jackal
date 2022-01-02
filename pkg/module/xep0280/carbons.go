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

	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
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
	hk     *hook.Hooks
}

// New returns a new initialized carbons instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	resMng *c2s.ResourceManager,
	hk *hook.Hooks,
) *Carbons {
	return &Carbons{
		hosts:  hosts,
		router: router,
		resMng: resMng,
		hk:     hk,
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
	p.hk.AddHook(hook.C2SStreamWillRouteElement, p.onC2SElementWillRoute, hook.DefaultPriority)
	p.hk.AddHook(hook.S2SInStreamWillRouteElement, p.onS2SElementWillRoute, hook.DefaultPriority)
	p.hk.AddHook(hook.C2SStreamMessageRouted, p.onC2SMessageRouted, hook.DefaultPriority)
	p.hk.AddHook(hook.S2SInStreamMessageRouted, p.onS2SMessageRouted, hook.DefaultPriority)

	log.Infow("started carbons module", "xep", XEPNumber)
	return nil
}

// Stop stops carbons module.
func (p *Carbons) Stop(_ context.Context) error {
	p.hk.RemoveHook(hook.C2SStreamWillRouteElement, p.onC2SElementWillRoute)
	p.hk.RemoveHook(hook.S2SInStreamWillRouteElement, p.onS2SElementWillRoute)
	p.hk.RemoveHook(hook.C2SStreamMessageRouted, p.onC2SMessageRouted)
	p.hk.RemoveHook(hook.S2SInStreamMessageRouted, p.onS2SMessageRouted)

	log.Infow("stopped carbons module", "xep", XEPNumber)
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

func (p *Carbons) onC2SElementWillRoute(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)

	msg, ok := inf.Element.(*stravaganza.Message)
	if !ok {
		return nil
	}
	inf.Element = stripMessagePrivate(msg)
	return nil
}

func (p *Carbons) onS2SElementWillRoute(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.S2SStreamInfo)

	msg, ok := inf.Element.(*stravaganza.Message)
	if !ok {
		return nil
	}
	inf.Element = stripMessagePrivate(msg)
	return nil
}

func (p *Carbons) onC2SMessageRouted(ctx context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)

	msg, ok := inf.Element.(*stravaganza.Message)
	if !ok {
		return nil
	}
	return p.processMessage(ctx, msg, inf.Targets)
}

func (p *Carbons) onS2SMessageRouted(ctx context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.S2SStreamInfo)

	msg, ok := inf.Element.(*stravaganza.Message)
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
		log.Infow("enabled carbons copy", "username", fromJID.Node(), "resource", fromJID.Resource())

	case iq.ChildNamespace("disable", carbonsNamespace) != nil:
		if err := p.setCarbonsEnabled(ctx, fromJID.Node(), fromJID.Resource(), false); err != nil {
			return err
		}
		log.Infow("disabled carbons copy", "username", fromJID.Node(), "resource", fromJID.Resource())

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
	return stm.SetInfoValue(ctx, carbonsEnabledCtxKey, enabled)
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
		if !res.Info().Bool(carbonsEnabledCtxKey) {
			continue
		}
		_, _ = p.router.Route(ctx, sentMsgCC(msg, res.JID()))
	}
	return nil
}

func (p *Carbons) routeReceivedCC(ctx context.Context, msg *stravaganza.Message, username string, ignoringTargets []jid.JID) error {
	rss, err := p.getFilteredResources(ctx, username, ignoringTargets)
	if err != nil {
		return err
	}
	for _, res := range rss {
		if !res.Info().Bool(carbonsEnabledCtxKey) {
			continue
		}
		_, _ = p.router.Route(ctx, receivedMsgCC(msg, res.JID()))
	}
	return nil
}

func (p *Carbons) getFilteredResources(ctx context.Context, username string, ignoringJIDs []jid.JID) ([]c2smodel.ResourceDesc, error) {
	rs, err := p.resMng.GetResources(ctx, username)
	if err != nil {
		return nil, err
	}
	ignoredJIDs := make(map[string]struct{}, len(ignoringJIDs))
	for _, j := range ignoringJIDs {
		ignoredJIDs[j.String()] = struct{}{}
	}
	var ret []c2smodel.ResourceDesc
	for _, res := range rs {
		_, ok := ignoredJIDs[res.JID().String()]
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

func stripMessagePrivate(msg *stravaganza.Message) *stravaganza.Message {
	if msg.ChildNamespace("private", carbonsNamespace) == nil {
		return msg
	}
	newMsg, _ := stravaganza.NewBuilderFromElement(msg).
		WithoutChildrenNamespace("private", carbonsNamespace).
		BuildMessage()
	return newMsg
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
