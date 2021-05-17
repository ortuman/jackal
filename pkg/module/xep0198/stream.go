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

package xep0198

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const (
	streamNamespace     = "urn:xmpp:sm:3"
	xmppStanzaNamespace = "urn:ietf:params:xml:ns:xmpp-stanzas"

	enabledInfoKey = "xep0198:enabled"

	badRequest        = "bad-request"
	unexpectedRequest = "unexpected-request"
)

const (
	// ModuleName represents stream module name.
	ModuleName = "stream_mgmt"

	// XEPNumber represents stream XEP number.
	XEPNumber = "0198"
)

// Config contains stream management module configuration options.
type Config struct {
	// AckTimeout defines stanza acknowledgement timeout.
	AckTimeout time.Time

	// MaxQueueSize defines maximum number of unacknowledged stanzas.
	// When the limit is reached, the c2s stream is terminated.
	MaxQueueSize int
}

// Stream represents a stream (XEP-0198) module type.
type Stream struct {
	router router.Router
	hosts  *host.Hosts
	hk     *hook.Hooks

	mu       sync.RWMutex
	managers map[string]*manager
}

// New returns a new initialized Stream instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	hk *hook.Hooks,
) *Stream {
	return &Stream{
		router:   router,
		hosts:    hosts,
		hk:       hk,
		managers: make(map[string]*manager),
	}
}

// Name returns stream module name.
func (m *Stream) Name() string { return ModuleName }

// StreamFeature returns stream module stream feature.
func (m *Stream) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return stravaganza.NewBuilder("sm").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build(), nil
}

// ServerFeatures returns stream server disco features.
func (m *Stream) ServerFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// AccountFeatures returns stream account disco features.
func (m *Stream) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts stream module.
func (m *Stream) Start(_ context.Context) error {
	m.hk.AddHook(hook.C2SStreamElementReceived, m.onElementRecv, hook.DefaultPriority)
	m.hk.AddHook(hook.C2SStreamElementSent, m.onElementSent, hook.DefaultPriority)

	log.Infow("Started stream module", "xep", XEPNumber)
	return nil
}

// Stop stops stream module.
func (m *Stream) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook.C2SStreamElementReceived, m.onElementRecv)
	m.hk.RemoveHook(hook.C2SStreamElementSent, m.onElementSent)

	log.Infow("Stopped stream module", "xep", XEPNumber)
	return nil
}

func (m *Stream) onElementRecv(ctx context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stm := execCtx.Sender.(stream.C2S)
	if inf.Element.Attribute(stravaganza.Namespace) == streamNamespace {
		return m.processCmd(ctx, inf.Element, stm)
	}
	_, ok := inf.Element.(stravaganza.Stanza)
	if !ok {
		return nil
	}
	m.processInboundStanza(stm)
	return nil
}

func (m *Stream) onElementSent(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stanza, ok := inf.Element.(stravaganza.Stanza)
	if !ok {
		return nil
	}
	m.processOutboundStanza(stanza, execCtx.Sender.(stream.C2S))
	return nil
}

func (m *Stream) processCmd(ctx context.Context, cmd stravaganza.Element, stm stream.C2S) error {
	if cmd.ChildrenCount() > 0 {
		sendFailedReply(badRequest, "Malformed element", stm)
		return hook.ErrStopped
	}
	if !stm.IsBinded() {
		sendFailedReply(unexpectedRequest, "", stm)
		return hook.ErrStopped
	}
	switch cmd.Name() {
	case "enable":
		return m.processEnable(ctx, stm)
	case "a":
		m.processA(stm, cmd.Attribute("h"))
	case "r":
		m.processR(stm)
	default:
		errText := fmt.Sprintf("Unknown tag %s qualified by namespace '%s'", cmd.Name(), streamNamespace)
		sendFailedReply(badRequest, errText, stm)
	}
	return hook.ErrStopped
}

func (m *Stream) processInboundStanza(stm stream.C2S) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mng := m.managers[streamID(stm)]
	if mng == nil {
		return
	}
	mng.processInboundStanza()
}

func (m *Stream) processOutboundStanza(stanza stravaganza.Stanza, stm stream.C2S) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mng := m.managers[streamID(stm)]
	if mng == nil {
		return
	}
	mng.processOutboundStanza(stanza)
}

func (m *Stream) processEnable(ctx context.Context, stm stream.C2S) error {
	enabled, _ := strconv.ParseBool(stm.Value(enabledInfoKey))
	if enabled {
		sendFailedReply(unexpectedRequest, "Stream management is already enabled", stm)
		return nil
	}
	if err := stm.SetValue(ctx, enabledInfoKey, "true"); err != nil {
		return err
	}
	mng, err := newManager(stm)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.managers[streamID(stm)] = mng
	m.mu.Unlock()

	stm.SendElement(stravaganza.NewBuilder("enabled").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build(),
	)
	log.Infow("Enabled stream management",
		"username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	return nil
}

func (m *Stream) processA(stm stream.C2S, h string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mng := m.managers[streamID(stm)]
	if mng == nil {
		return
	}
	hVal, _ := strconv.ParseUint(h, 10, 32)
	if hVal == 0 {
		return
	}
	mng.acknowledge(uint32(hVal))

	log.Infow("Received stanza ack",
		"ack_h", hVal, "h", mng.outboundH(), "username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	pending := mng.queue()
	if len(pending) == 0 {
		return // done here
	}
	log.Infow("Resending pending stanzas...",
		"len", len(pending), "username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	for _, stanza := range pending {
		stm.SendElement(stanza)
	}
}

func (m *Stream) processR(stm stream.C2S) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mng := m.managers[streamID(stm)]
	if mng == nil {
		return
	}
	log.Infow("Stanza ack requested",
		"username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	a := stravaganza.NewBuilder("a").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("h", strconv.FormatUint(uint64(mng.inboundH()), 10)).
		Build()
	stm.SendElement(a)
}

func streamID(stm stream.C2S) string {
	return fmt.Sprintf("%s/%s", stm.Username(), stm.Resource())
}

func encodeSMID(jd *jid.JID, nonce []byte) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(jd.String())
	buf.WriteByte(0)
	buf.Write(nonce)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodeSMID(smID string) (jd *jid.JID, nonce []byte, err error) {
	return nil, nil, nil
}

func sendFailedReply(reason string, text string, stm stream.C2S) {
	sb := stravaganza.NewBuilder("failed").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithChild(
			stravaganza.NewBuilder(reason).
				WithAttribute(stravaganza.Namespace, xmppStanzaNamespace).
				Build(),
		)
	if len(text) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("text").
				WithAttribute(stravaganza.Namespace, xmppStanzaNamespace).
				WithText(text).
				Build(),
		)
	}
	_ = stm.SendElement(sb.Build())
}
