// Copyright 2022 The jackal Authors
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

	"github.com/ortuman/jackal/pkg/cluster/instance"

	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"

	streamqueue "github.com/ortuman/jackal/pkg/module/xep0198/queue"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/cluster/resourcemanager"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
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
	itemNotFound      = "item-not-found"

	nonceLength = 24

	// unacknowledgedStanzaCount defines the stanza count interval at which an "r" stanza will be sent
	unacknowledgedStanzaCount = 25
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
	// HibernateTime defines the amount of time a stream
	// can stay in disconnected state before being terminated.
	HibernateTime time.Duration `fig:"hibernate_time" default:"3m"`

	// RequestAckInterval defines the period of stream inactivity
	// that should be waited before requesting acknowledgement.
	RequestAckInterval time.Duration `fig:"request_ack_interval" default:"1m"`

	// WaitForAckTimeout defines stanza acknowledgement timeout.
	WaitForAckTimeout time.Duration `fig:"wait_for_ack_timeout" default:"30s"`

	// MaxQueueSize defines maximum number of unacknowledged stanzas.
	// When the limit is reached the c2s stream is terminated.
	MaxQueueSize int `fig:"max_queue_size" default:"250"`
}

// Stream represents a stream (XEP-0198) module type.
type Stream struct {
	cfg    Config
	router router.Router
	hosts  *host.Hosts
	resMng resourcemanager.Manager
	hk     *hook.Hooks
	logger kitlog.Logger

	stmQueueMap    *streamqueue.QueueMap
	clusterConnMng clusterConnManager

	mu      sync.RWMutex
	termTms map[string]*time.Timer
}

// New returns a new initialized Stream instance.
func New(
	cfg Config,
	stmQueueMap *streamqueue.QueueMap,
	clusterConnMng *clusterconnmanager.Manager,
	router router.Router,
	hosts *host.Hosts,
	resMng resourcemanager.Manager,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *Stream {
	return &Stream{
		cfg:            cfg,
		router:         router,
		hosts:          hosts,
		resMng:         resMng,
		stmQueueMap:    stmQueueMap,
		clusterConnMng: clusterConnMng,
		termTms:        make(map[string]*time.Timer),
		hk:             hk,
		logger:         kitlog.With(logger, "module", ModuleName, "xep", XEPNumber),
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

	level.Info(m.logger).Log("msg", "started stream module")
	return nil
}

// Stop stops stream module.
func (m *Stream) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook.C2SStreamElementReceived, m.onElementRecv)
	m.hk.RemoveHook(hook.C2SStreamElementSent, m.onElementSent)
	m.hk.RemoveHook(hook.C2SStreamDisconnected, m.onDisconnect)
	m.hk.RemoveHook(hook.C2SStreamTerminated, m.onTerminate)

	level.Info(m.logger).Log("msg", "stopped stream module")
	return nil
}

func (m *Stream) onElementRecv(execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	ctx := execCtx.Context

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
	sq := m.stmQueueMap.Get(queueKey(stm.JID()))
	if sq == nil {
		return nil
	}
	sq.HandleIn()
	return nil
}

func (m *Stream) onElementSent(execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stanza, ok := inf.Element.(stravaganza.Stanza)
	if !ok {
		return nil
	}
	stm := execCtx.Sender.(stream.C2S)

	sq := m.stmQueueMap.Get(queueKey(stm.JID()))
	if sq == nil {
		return nil
	}
	sq.HandleOut(stanza)

	qLen := sq.Len()
	switch {
	case qLen >= m.cfg.MaxQueueSize:
		_ = sq.GetStream().Disconnect(streamerror.E(streamerror.PolicyViolation))

		level.Info(m.logger).Log("msg", "max queue size reached",
			"id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
		)

	case qLen%unacknowledgedStanzaCount == 0:
		sq.RequestAck()
	}
	return nil
}

func (m *Stream) onDisconnect(execCtx *hook.ExecutionContext) error {
	stm := execCtx.Sender.(stream.C2S)
	if !stm.Info().Bool(enabledInfoKey) {
		return nil
	}
	sq := m.stmQueueMap.Get(queueKey(stm.JID()))
	if sq == nil {
		return nil
	}
	// cancel scheduled timers
	sq.CancelTimers()

	inf := execCtx.Info.(*hook.C2SStreamInfo)
	discErr := inf.DisconnectError
	_, ok := discErr.(*streamerror.Error)
	if ok || errors.Is(discErr, xmppparser.ErrStreamClosedByPeer) {
		return nil
	}
	// schedule stream termination
	m.mu.Lock()
	m.termTms[inf.ID] = time.AfterFunc(m.cfg.HibernateTime, func() {
		_ = stm.Disconnect(nil)

		level.Info(m.logger).Log("msg", "hibernated stream terminated",
			"id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
		)
	})
	m.mu.Unlock()

	level.Info(m.logger).Log("msg", "scheduled stream termination",
		"id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	return hook.ErrStopped
}

func (m *Stream) onTerminate(execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	stm := execCtx.Sender.(stream.C2S)
	if !stm.Info().Bool(enabledInfoKey) {
		return nil
	}
	// unregister stream queue
	qk := queueKey(stm.JID())

	sq := m.stmQueueMap.Get(qk)
	if sq == nil {
		return nil
	}
	m.stmQueueMap.Delete(qk)

	// cancel scheduled termination
	m.mu.Lock()
	if tm := m.termTms[inf.ID]; tm != nil {
		tm.Stop()
	}
	delete(m.termTms, inf.ID)
	m.mu.Unlock()

	return nil
}

func (m *Stream) processCmd(ctx context.Context, cmd stravaganza.Element, stm stream.C2S) error {
	if cmd.ChildrenCount() > 0 {
		sendFailedReply(badRequest, "Malformed element", stm)
		return nil
	}
	h, _ := strconv.ParseUint(cmd.Attribute("h"), 10, 32)

	switch cmd.Name() {
	case "enable":
		return m.handleEnable(ctx, stm)
	case "resume":
		prevID := cmd.Attribute("previd")
		return m.handleResume(ctx, stm, uint32(h), prevID)
	case "a":
		m.handleA(stm, uint32(h))
	case "r":
		m.handleR(stm)
	default:
		errText := fmt.Sprintf("Unknown tag %s qualified by namespace '%s'", cmd.Name(), streamNamespace)
		sendFailedReply(badRequest, errText, stm)
	}
	return nil
}

func (m *Stream) handleEnable(ctx context.Context, stm stream.C2S) error {
	if !stm.IsBinded() {
		sendFailedReply(unexpectedRequest, "", stm)
		return nil
	}
	if stm.Info().Bool(enabledInfoKey) {
		sendFailedReply(unexpectedRequest, "Stream management is already enabled", stm)
		return nil
	}
	if err := stm.SetInfoValue(ctx, enabledInfoKey, true); err != nil {
		return err
	}
	// generate nonce
	nonce := make([]byte, nonceLength)
	for i := range nonce {
		nonce[i] = byte(rand.Intn(255) + 1)
	}
	// register stream queue
	sq := streamqueue.New(
		stm,
		nonce,
		nil,
		0,
		0,
		m.cfg.RequestAckInterval,
		m.cfg.WaitForAckTimeout,
	)
	m.stmQueueMap.Set(queueKey(stm.JID()), sq)

	smID := encodeSMID(stm.JID(), nonce)

	stm.SendElement(stravaganza.NewBuilder("enabled").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("id", smID).
		WithAttribute("resume", "true").
		Build(),
	)
	level.Info(m.logger).Log("msg", "enabled stream management",
		"smID", smID, "id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	return nil
}

func (m *Stream) handleResume(ctx context.Context, stm stream.C2S, h uint32, prevSMID string) error {
	if !stm.IsAuthenticated() {
		sendFailedReply(unexpectedRequest, "", stm)
		return nil
	}
	// perform stream resumption
	jd, nonce, err := decodeSMID(prevSMID)
	if err != nil {
		return err
	}
	// fetch resource info
	res, err := m.resMng.GetResource(ctx, jd.Node(), jd.Resource())
	if err != nil {
		return err
	}
	if res == nil {
		sendFailedReply(itemNotFound, "", stm)
		return nil
	}
	var sq *streamqueue.Queue

	qk := queueKey(jd)

	if res.InstanceID() == instance.ID() { // local retained queue
		sq = m.stmQueueMap.Get(qk)
		if sq == nil {
			sendFailedReply(itemNotFound, "", stm)
			return nil
		}
		// disconnect hibernated c2s stream
		if err := <-sq.GetStream().Disconnect(streamerror.E(streamerror.Conflict)); err != nil {
			return err
		}
		// set new stream
		sq.SetStream(stm)

	} else { // transfer retained queue from internal cluster instance
		conn, err := m.clusterConnMng.GetConnection(res.InstanceID())
		if err != nil {
			return err
		}
		resp, err := conn.StreamManagement().TransferQueue(ctx, qk)
		if err != nil {
			return err
		}
		sq = streamqueue.New(
			stm,
			resp.Nonce,
			resp.Elements,
			resp.InH,
			resp.OutH,
			m.cfg.RequestAckInterval,
			m.cfg.WaitForAckTimeout,
		)

		level.Info(m.logger).Log(
			"msg", "stream queue transferred", "key", qk, "from", res.InstanceID(), "to", instance.ID(),
		)
	}

	// invalid smID?
	if !jd.MatchesWithOptions(stm.JID(), jid.MatchesBare) || bytes.Compare(sq.Nonce(), nonce) != 0 {
		sendFailedReply(itemNotFound, "", stm)
		return nil
	}

	// register retained queue
	m.stmQueueMap.Set(qk, sq)

	// resume stream and send unacknowledged stanzas
	if err := stm.Resume(ctx, res.JID(), res.Presence(), res.Info()); err != nil {
		return err
	}
	stm.SendElement(stravaganza.NewBuilder("resumed").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("h", strconv.FormatUint(uint64(sq.InboundH()), 10)).
		WithAttribute("previd", prevSMID).
		Build(),
	)
	sq.Acknowledge(h)
	sq.SendPending()
	sq.ScheduleR()

	level.Info(m.logger).Log("msg", "resumed stream",
		"smID", prevSMID, "id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	return nil
}

func (m *Stream) handleA(stm stream.C2S, h uint32) {
	sq := m.stmQueueMap.Get(queueKey(stm.JID()))
	if sq == nil {
		return
	}
	sq.Acknowledge(h)

	level.Info(m.logger).Log("msg", "received stanza ack",
		"ack_h", h, "h", sq.OutboundH(), "id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	if sq.Len() == 0 {
		return // done here
	}
	level.Info(m.logger).Log("msg", "resending pending stanzas...",
		"len", sq.Len(), "id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	sq.SendPending()
}

func (m *Stream) handleR(stm stream.C2S) {
	sq := m.stmQueueMap.Get(queueKey(stm.JID()))
	if sq == nil {
		return
	}
	level.Info(m.logger).Log("msg", "stanza ack requested",
		"id", stm.ID(), "username", stm.Username(), "resource", stm.Resource(),
	)
	a := stravaganza.NewBuilder("a").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		WithAttribute("h", strconv.FormatUint(uint64(sq.InboundH()), 10)).
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

func queueKey(jd *jid.JID) string {
	return jd.String()
}
