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

package xep0114

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/runqueue/v2"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func init() {
	disconnectTimeout = time.Second
}

func TestInComponent_SendStanza(t *testing.T) {
	// given
	sessMock := &sessionMock{}

	var mtx sync.RWMutex
	sendBuf := bytes.NewBuffer(nil)

	sessMock.SendFunc = func(ctx context.Context, element stravaganza.Element) error {
		mtx.Lock()
		defer mtx.Unlock()
		_ = element.ToXML(sendBuf, true)
		return nil
	}
	s := &inComponent{
		session: sessMock,
		rq:      runqueue.New("in_component:test"),
		hk:      hook.NewHooks(),
		logger:  kitlog.NewNopLogger(),
	}
	// when
	s.sendStanza(testMessageStanza())

	time.Sleep(time.Millisecond * 250)

	// then
	mtx.Lock()
	defer mtx.Unlock()

	require.Equal(t, `<message from='noelia@jackal.im/yard' to='ortuman@jackal.im/balcony'><body>I&#39;ll give thee a wind.</body></message>`, sendBuf.String())
}

func TestInComponent_Shutdown(t *testing.T) {
	// given
	trMock := &transportMock{}
	compsMock := &componentsMock{}
	extCompMngMock := &externalComponentManagerMock{}
	sessMock := &sessionMock{}

	trMock.CloseFunc = func() error { return nil }
	compsMock.UnregisterComponentFunc = func(ctx context.Context, cHost string) error {
		return nil
	}
	extCompMngMock.UnregisterComponentHostFunc = func(ctx context.Context, cHost string) error {
		return nil
	}

	var mtx sync.RWMutex

	sendBuf := bytes.NewBuffer(nil)
	sessMock.SendFunc = func(ctx context.Context, element stravaganza.Element) error {
		mtx.Lock()
		defer mtx.Unlock()

		_ = element.ToXML(sendBuf, true)
		return nil
	}
	sessMock.CloseFunc = func(ctx context.Context) error { return nil }

	s := &inComponent{
		state:      uint32(authenticated),
		session:    sessMock,
		tr:         trMock,
		comps:      compsMock,
		extCompMng: extCompMngMock,
		inHub:      newInHub(),
		hk:         hook.NewHooks(),
		logger:     kitlog.NewNopLogger(),
		rq:         runqueue.New("in_component:test"),
		doneCh:     make(chan struct{}),
	}
	// when
	s.shutdown()

	time.Sleep(disconnectTimeout + time.Second) // wait for disconnect

	// then
	mtx.Lock()
	defer mtx.Unlock()

	require.Equal(t, `<stream:error><system-shutdown xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/></stream:error>`, sendBuf.String())
	require.Len(t, sessMock.CloseCalls(), 1)
	require.Len(t, trMock.CloseCalls(), 1)
}

func TestInComponent_HandleSessionElement(t *testing.T) {
	b := stravaganza.NewIQBuilder()
	b.WithAttribute("id", "iq-1234")
	b.WithAttribute("type", "get")
	b.WithAttribute("from", "upload.localhost")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("query").
			WithAttribute(stravaganza.Namespace, "Hi there!").
			Build(),
	)
	iq, _ := b.BuildIQ()

	var tests = []struct {
		name string

		// input
		state        inComponentState
		sessionResFn func() (stravaganza.Element, error)
		routeError   error

		// expectations
		expectedOutput string
		expectRouted   bool
		expectedState  inComponentState
	}{
		{
			name:  "Connecting",
			state: connecting,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:component:accept").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.To, "upload.localhost").
					Build(), nil
			},
			expectedOutput: `<?xml version='1.0'?><stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='upload.localhost' id='comp-1'>`,
			expectedState:  handshaking,
		},
		{
			name:  "Handshaking/Success",
			state: handshaking,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("handshake").
					WithText("66feed75b630cad7f6422be95dc40976222c5cca").
					Build(), nil
			},
			expectedOutput: `<handshake/>`,
			expectedState:  authenticated,
		},
		{
			name:  "Handshaking/Fail",
			state: handshaking,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("handshake").
					WithText("foo").
					Build(), nil
			},
			expectedOutput: `<stream:error><not-authorized xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/></stream:error></stream:stream>`,
			expectedState:  disconnected,
		},
		{
			name:  "Route",
			state: authenticated,
			sessionResFn: func() (stravaganza.Element, error) {
				return iq, nil
			},
			expectedState: authenticated,
			expectRouted:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ssMock := &sessionMock{}
			trMock := &transportMock{}
			compsMock := &componentsMock{}
			extCompMngMock := &externalComponentManagerMock{}
			routerMock := &routerMock{}

			trMock.SetReadRateLimiterFunc = func(rLim *rate.Limiter) error { return nil }
			trMock.CloseFunc = func() error {
				return nil
			}

			outBuf := bytes.NewBuffer(nil)

			openStr := `<?xml version='1.0'?><stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='upload.localhost' id='comp-1'>`
			ssMock.OpenComponentFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader(openStr))
				return err
			}
			ssMock.CloseFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("</stream:stream>"))
				return err
			}
			ssMock.StreamIDFunc = func() string {
				return "comp-1"
			}
			ssMock.SetFromJIDFunc = func(_ *jid.JID) {}
			ssMock.ResetFunc = func(_ transport.Transport) error { return nil }
			ssMock.SendFunc = func(_ context.Context, element stravaganza.Element) error {
				return element.ToXML(outBuf, true)
			}

			compsMock.IsComponentHostFunc = func(cHost string) bool { return false }
			compsMock.RegisterComponentFunc = func(ctx context.Context, compo component.Component) error {
				return nil
			}
			compsMock.UnregisterComponentFunc = func(ctx context.Context, cHost string) error {
				return nil
			}
			extCompMngMock.RegisterComponentHostFunc = func(ctx context.Context, cHost string) error {
				return nil
			}
			extCompMngMock.UnregisterComponentHostFunc = func(ctx context.Context, cHost string) error {
				return nil
			}

			var routed bool
			routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
				routed = true
				return nil, nil
			}

			stm := &inComponent{
				cfg: inConfig{
					reqTimeout:    time.Minute,
					maxStanzaSize: 8192,
					secret:        "a-secret-1",
				},
				state:      uint32(tt.state),
				rq:         runqueue.New(tt.name),
				doneCh:     make(chan struct{}),
				tr:         trMock,
				session:    ssMock,
				router:     routerMock,
				comps:      compsMock,
				extCompMng: extCompMngMock,
				inHub:      newInHub(),
				hk:         hook.NewHooks(),
				logger:     kitlog.NewNopLogger(),
			}
			// when
			stm.handleSessionResult(tt.sessionResFn())

			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectedState, stm.getState())
			require.Equal(t, tt.expectRouted, routed)
		})
	}
}

func TestInComponent_HandleSessionError(t *testing.T) {
	var tests = []struct {
		name           string
		state          inComponentState
		sErr           error
		expectedOutput string
		expectClosed   bool
	}{
		{
			name:           "ClosedByPeerError",
			state:          authenticated,
			sErr:           xmppparser.ErrStreamClosedByPeer,
			expectedOutput: `</stream:stream>`,
			expectClosed:   true,
		},
		{
			name:           "EOFError",
			state:          authenticated,
			sErr:           io.EOF,
			expectedOutput: ``,
			expectClosed:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ssMock := &sessionMock{}
			trMock := &transportMock{}
			compsMock := &componentsMock{}
			extCompMngMock := &externalComponentManagerMock{}
			routerMock := &routerMock{}

			outBuf := bytes.NewBuffer(nil)
			ssMock.OpenComponentFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("<stream:stream>"))
				return err
			}
			ssMock.CloseFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("</stream:stream>"))
				return err
			}
			ssMock.SendFunc = func(_ context.Context, element stravaganza.Element) error {
				return element.ToXML(outBuf, true)
			}

			var trClosed bool
			trMock.CloseFunc = func() error {
				trClosed = true
				return nil
			}

			compsMock.RegisterComponentFunc = func(ctx context.Context, compo component.Component) error {
				return nil
			}
			compsMock.UnregisterComponentFunc = func(ctx context.Context, cHost string) error {
				return nil
			}
			extCompMngMock.RegisterComponentHostFunc = func(ctx context.Context, cHost string) error {
				return nil
			}
			extCompMngMock.UnregisterComponentHostFunc = func(ctx context.Context, cHost string) error {
				return nil
			}

			routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
				return nil, nil
			}

			stm := &inComponent{
				cfg: inConfig{
					reqTimeout:    time.Minute,
					maxStanzaSize: 8192,
				},
				state:      uint32(tt.state),
				rq:         runqueue.New(tt.name),
				doneCh:     make(chan struct{}),
				tr:         trMock,
				session:    ssMock,
				router:     routerMock,
				comps:      compsMock,
				extCompMng: extCompMngMock,
				inHub:      newInHub(),
				hk:         hook.NewHooks(),
				logger:     kitlog.NewNopLogger(),
			}
			// when
			stm.handleSessionResult(nil, tt.sErr)

			// then
			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectClosed, trClosed)
		})
	}
}

func testMessageStanza() *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()
	return msg
}
