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
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2/jid"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const (
	streamNamespace     = "urn:xmpp:sm:3"
	xmppStanzaNamespace = "urn:ietf:params:xml:ns:xmpp-stanzas"

	enabledInfoKey = "xep0198:enabled"

	badRequest        = "bad-request"
	unexpectedRequest = "unexpected-request"

	nonceLength = 24
)

var errInvalidSMID = errors.New("xep0198: invalid stream identifier format")

const (
	// ModuleName represents stream module name.
	ModuleName = "stream_mgmt"

	// XEPNumber represents stream XEP number.
	XEPNumber = "0198"
)

// Config contains stream management module configuration options.
type Config struct {
	// HibernateTime defines defines the amount of time a stream
	// can stay in disconnected state before being terminated.
	HibernateTime time.Duration

	// AckTimeout defines stanza acknowledgement timeout.
	AckTimeout time.Duration

	// MaxQueueSize defines maximum number of unacknowledged stanzas.
	// When the limit is reached, the c2s stream is terminated.
	MaxQueueSize int
}

// Stream represents a stream (XEP-0198) module type.
type Stream struct {
	cfg    Config
	router router.Router
	hosts  *host.Hosts
	hk     *hook.Hooks

	mu         sync.RWMutex
	queues     map[string]*stmQ
	termTimers map[string]*time.Timer
}

// New returns a new initialized Stream instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	hk *hook.Hooks,
	cfg Config,
) *Stream {
	return &Stream{
		cfg:        cfg,
		router:     router,
		hosts:      hosts,
		hk:         hk,
		queues:     make(map[string]*stmQ),
		termTimers: make(map[string]*time.Timer),
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
	m.hk.AddHook(hook.C2SStreamDisconnected, m.onDisconnect, hook.LowestPriority)
	m.hk.AddHook(hook.C2SStreamTerminated, m.onTerminate, hook.LowestPriority)

	log.Infow("Started stream module", "xep", XEPNumber)
	return nil
}

// Stop stops stream module.
func (m *Stream) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook.C2SStreamElementReceived, m.onElementRecv)
	m.hk.RemoveHook(hook.C2SStreamElementSent, m.onElementSent)
	m.hk.RemoveHook(hook.C2SStreamDisconnected, m.onDisconnect)
	m.hk.RemoveHook(hook.C2SStreamTerminated, m.onTerminate)

	log.Infow("Stopped stream module", "xep", XEPNumber)
	return nil
}

func (m *Stream) onElementRecv(ctx context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stm := execCtx.Sender.(stream.C2S)
	if inf.Element.Attribute(stravaganza.Namespace) == streamNamespace {
		if err := m.processCmd(ctx, inf.Element, stm); err != nil {
			return err
		}
		return hook.ErrStopped // already handled
	}
	_, ok := inf.Element.(stravaganza.Stanza)
	if !ok {
		return nil
	}
	m.mu.RLock()
	sq := m.queues[stmID(stm)]
	m.mu.RUnlock()
	if sq == nil {
		return nil
	}
	sq.processInboundStanza()
	return nil
}

func (m *Stream) onElementSent(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stanza, ok := inf.Element.(stravaganza.Stanza)
	if !ok {
		return nil
	}
	stm := execCtx.Sender.(stream.C2S)

	m.mu.RLock()
	sq := m.queues[stmID(stm)]
	m.mu.RUnlock()
	if sq == nil {
		return nil
	}
	sq.processOutboundStanza(stanza)
	return nil
}

func (m *Stream) onDisconnect(_ context.Context, execCtx *hook.ExecutionContext) error {
	stm := execCtx.Sender.(stream.C2S)

	m.mu.Lock()
	defer m.mu.Unlock()

	sq := m.queues[stmID(stm)]
	if sq == nil {
		return nil
	}
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	discErr := inf.DisconnectError
	_, ok := discErr.(*streamerror.Error)
	if ok || errors.Is(discErr, xmppparser.ErrStreamClosedByPeer) {
		return nil
	}
	// cancel scheduled R
	sq.cancelR()

	// schedule stream termination
	m.termTimers[inf.ID] = time.AfterFunc(m.cfg.HibernateTime, func() {
		_ = stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))

		log.Infow("Hibernated stream terminated",
			"username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
		)
	})

	log.Infow("Scheduled stream termination",
		"username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	return hook.ErrStopped
}

func (m *Stream) onTerminate(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stm := execCtx.Sender.(stream.C2S)

	// unregister stream queue
	m.mu.Lock()
	defer m.mu.Unlock()

	sID := stmID(stm)
	sq := m.queues[sID]
	if sq == nil {
		return nil
	}
	sq.cancelTimers()
	delete(m.queues, sID)

	// cancel scheduled termination
	if tm := m.termTimers[inf.ID]; tm != nil {
		tm.Stop()
	}
	delete(m.termTimers, inf.ID)

	return nil
}

func (m *Stream) processCmd(ctx context.Context, cmd stravaganza.Element, stm stream.C2S) error {
	if cmd.ChildrenCount() > 0 {
		sendFailedReply(badRequest, "Malformed element", stm)
		return nil
	}
	if !stm.IsBinded() {
		sendFailedReply(unexpectedRequest, "", stm)
		return nil
	}
	h := cmd.Attribute("h")

	switch cmd.Name() {
	case "enable":
		return m.handleEnable(ctx, stm)
	case "resume":
		prevID := cmd.Attribute("previd")
		return m.handleResume(ctx, stm, h, prevID)
	case "a":
		m.handleA(stm, h)
	case "r":
		m.handleR(stm)
	default:
		errText := fmt.Sprintf("Unknown tag %s qualified by namespace '%s'", cmd.Name(), streamNamespace)
		sendFailedReply(badRequest, errText, stm)
	}
	return nil
}

func (m *Stream) handleEnable(ctx context.Context, stm stream.C2S) error {
	if stm.Info().Bool(enabledInfoKey) {
		sendFailedReply(unexpectedRequest, "Stream management is already enabled", stm)
		return nil
	}
	if err := stm.SetInfoValue(ctx, enabledInfoKey, true); err != nil {
		return err
	}
	// register stream queue
	m.mu.Lock()
	defer m.mu.Unlock()

	// generate nonce
	nonce := make([]byte, nonceLength)
	for i := range nonce {
		nonce[i] = byte(rand.Intn(255) + 1)
	}
	m.queues[stmID(stm)] = newSQ(stm, nonce)

	smID := encodeSMID(stm.JID(), nonce)

	stm.SendElement(stravaganza.NewBuilder("enabled").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("id", smID).
		WithAttribute("location", instance.Hostname()).
		WithAttribute("resume", "true").
		Build(),
	)
	log.Infow("Enabled stream management",
		"smid", smID, "username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	return nil
}

func (m *Stream) handleResume(ctx context.Context, stm stream.C2S, h, prevID string) error {
	// TODO(ortuman): implement resume logic
	return nil
}

func (m *Stream) handleA(stm stream.C2S, h string) {
	m.mu.RLock()
	sq := m.queues[stmID(stm)]
	m.mu.RUnlock()
	if sq == nil {
		return
	}
	hVal, _ := strconv.ParseUint(h, 10, 32)
	if hVal == 0 {
		return
	}
	sq.acknowledge(uint32(hVal))

	log.Infow("Received stanza ack",
		"ack_h", hVal, "h", sq.outboundH(), "username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	pending := sq.stanzas()
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

func (m *Stream) handleR(stm stream.C2S) {
	m.mu.RLock()
	sq := m.queues[stmID(stm)]
	m.mu.RUnlock()
	if sq == nil {
		return
	}
	log.Infow("Stanza ack requested",
		"username", stm.Username(), "resource", stm.Resource(), "xep", XEPNumber,
	)
	a := stravaganza.NewBuilder("a").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("h", strconv.FormatUint(uint64(sq.inboundH()), 10)).
		Build()
	stm.SendElement(a)
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

func encodeSMID(jd *jid.JID, nonce []byte) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(jd.String())
	buf.WriteByte(0)
	buf.Write(nonce)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodeSMID(smID string) (jd *jid.JID, nonce []byte, err error) {
	b, err := base64.StdEncoding.DecodeString(smID)
	if err != nil {
		return nil, nil, err
	}
	ss := bytes.Split(b, []byte{0})
	if len(ss) != 2 {
		return nil, nil, errInvalidSMID
	}
	jd, err = jid.NewWithString(string(ss[0]), false)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errInvalidSMID, err)
	}
	return jd, ss[1], nil
}

func stmID(stm stream.C2S) string {
	return fmt.Sprintf("%s/%s", stm.Username(), stm.Resource())
}
