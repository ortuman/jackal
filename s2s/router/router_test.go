/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

type mockedOutS2S struct {
	sentTimes int32
}

func (s *mockedOutS2S) ID() string                            { return "out-test-1" }
func (s *mockedOutS2S) Disconnect(_ context.Context, _ error) {}
func (s *mockedOutS2S) SendElement(_ context.Context, _ xmpp.XElement) {
	atomic.AddInt32(&s.sentTimes, 1)
}

type mockedOutProvider struct {
	outStm *mockedOutS2S
}

func (p *mockedOutProvider) GetOut(_, _ string) stream.S2SOut { return p.outStm }
func (p *mockedOutProvider) Shutdown(_ context.Context) error { return nil }

func TestS2SRouter_Route(t *testing.T) {
	outStm := &mockedOutS2S{}
	p := &mockedOutProvider{outStm: outStm}

	r := New(p)

	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	j2, _ := jid.NewWithString("noelia@jabber.org/yard", true)
	j3, _ := jid.NewWithString("ortuman@jabber.org/chamber", true)

	_ = r.Route(context.Background(), xmpp.NewPresence(j1, j2, xmpp.AvailableType), "jackal.im")
	_ = r.Route(context.Background(), xmpp.NewPresence(j1, j3, xmpp.AvailableType), "jackal.im")

	require.Equal(t, int32(2), atomic.LoadInt32(&outStm.sentTimes))
}
