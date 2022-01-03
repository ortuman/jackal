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

package s2s

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/jackal-xmpp/runqueue/v2"
	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/hook"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestOutS2S_SendElement(t *testing.T) {
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
	s := &outS2S{
		state:   outAuthenticated,
		session: sessMock,
		rq:      runqueue.New("out_s2s:test"),
		hk:      hook.NewHooks(),
		logger:  kitlog.NewNopLogger(),
	}
	// when
	stanza := stravaganza.NewBuilder("auth").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		Build()

	s.SendElement(stanza)

	time.Sleep(time.Millisecond * 250)

	// then
	mtx.Lock()
	defer mtx.Unlock()

	require.Equal(t, `<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`, sendBuf.String())
}

func TestOutS2S_Disconnect(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.CloseFunc = func() error { return nil }

	sessMock := &sessionMock{}

	var mtx sync.RWMutex

	sendBuf := bytes.NewBuffer(nil)
	sessMock.SendFunc = func(ctx context.Context, element stravaganza.Element) error {
		mtx.Lock()
		defer mtx.Unlock()

		_ = element.ToXML(sendBuf, true)
		return nil
	}
	sessMock.CloseFunc = func(ctx context.Context) error { return nil }

	s := &outS2S{
		state:   outAuthenticated,
		session: sessMock,
		tr:      trMock,
		rq:      runqueue.New("out_s2s:test"),
		hk:      hook.NewHooks(),
		logger:  kitlog.NewNopLogger(),
	}
	// when
	s.Disconnect(streamerror.E(streamerror.SystemShutdown))

	time.Sleep(time.Millisecond * 250) // wait for disconnect

	// then
	mtx.Lock()
	defer mtx.Unlock()

	require.Equal(t, `<stream:error><system-shutdown xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></stream:error>`, sendBuf.String())
	require.Len(t, sessMock.CloseCalls(), 1)
	require.Len(t, trMock.CloseCalls(), 1)
}

func TestOutS2S_HandleSessionElement(t *testing.T) {
	var tests = []struct {
		name string

		// input
		state        outState
		sender       string
		target       string
		sessionResFn func() (stravaganza.Element, error)
		flags        uint8

		// expectations
		expectedOutput string
		expectedState  outState
		expectedFlags  uint8
	}{
		{
			name:  "Connecting/Unsecured",
			state: outConnecting,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:server").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.To, "localhost").
					WithAttribute(stravaganza.Version, "1.0").
					Build(), nil
			},
			expectedOutput: ``,
			expectedState:  outConnected,
		},
		{
			name:  "Connected/StartTLS",
			state: outConnected,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:features").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithChild(
						stravaganza.NewBuilder("starttls").
							WithAttribute(stravaganza.Namespace, tlsNamespace).
							Build(),
					).
					Build(), nil
			},
			expectedOutput: `<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`,
			expectedState:  outSecuring,
		},
		{
			name:  "Securing/ProceedTLS",
			state: outSecuring,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("proceed").
					WithAttribute(stravaganza.Namespace, tlsNamespace).
					Build(), nil
			},
			expectedOutput: `<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" from="jackal.im" to="jabber.org" version="1.0">`,
			expectedState:  outConnecting,
			expectedFlags:  fSecured,
		},
		{
			name:  "Connected/Authenticate",
			state: outConnected,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:features").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithChild(
						stravaganza.NewBuilder("mechanisms").
							WithAttribute(stravaganza.Namespace, saslNamespace).
							WithChild(
								stravaganza.NewBuilder("mechanism").
									WithText("EXTERNAL").
									Build(),
							).
							Build(),
					).
					Build(), nil
			},
			expectedOutput: `<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="EXTERNAL">amFja2FsLmlt</auth>`,
			expectedState:  outAuthenticating,
		},
		{
			name:  "Authenticating/Success",
			state: outAuthenticating,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("success").
					WithAttribute(stravaganza.Namespace, saslNamespace).
					Build(), nil
			},
			expectedOutput: `<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" from="jackal.im" to="jabber.org" version="1.0">`,
			expectedFlags:  fSecured | fAuthenticated,
			expectedState:  outConnecting,
		},
		{
			name:  "Authenticating/Failed",
			state: outAuthenticating,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("failure").
					WithAttribute(stravaganza.Namespace, saslNamespace).
					Build(), nil
			},
			expectedOutput: `<stream:error><remote-connection-failed xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></stream:error></stream:stream>`,
			expectedState:  outDisconnected,
		},
		{
			name:  "Connected/Dialback",
			state: outConnected,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:features").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithChild(
						stravaganza.NewBuilder("dialback").
							WithAttribute(stravaganza.Namespace, dialbackNamespace).
							Build(),
					).
					Build(), nil
			},
			expectedOutput: `<db:result from="jackal.im" to="jabber.org">21bd4eb62f7d70d22b545f38a40a023ad6fa385905f36d889612fcb4cdb4966c</db:result>`,
			expectedState:  outVerifyingDialbackKey,
		},
		{
			name:  "VerifyingDialback/Valid",
			state: outVerifyingDialbackKey,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("db:result").
					WithAttribute(stravaganza.Type, "valid").
					WithAttribute(stravaganza.From, "jabber.org").
					WithAttribute(stravaganza.To, "jackal.im").
					Build(), nil
			},
			expectedOutput: ``,
			expectedState:  outAuthenticated,
		},
		{
			name:  "VerifyingDialback/Invalid",
			state: outVerifyingDialbackKey,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("db:result").
					WithAttribute(stravaganza.Type, "invalid").
					WithAttribute(stravaganza.From, "jabber.org").
					WithAttribute(stravaganza.To, "jackal.im").
					Build(), nil
			},
			expectedOutput: `<stream:error><remote-connection-failed xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></stream:error></stream:stream>`,
			expectedState:  outDisconnected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ssMock := &sessionMock{}
			trMock := &transportMock{}

			kvMock := &kvMock{}
			kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
				return nil
			}

			outBuf := bytes.NewBuffer(nil)
			ssMock.StreamIDFunc = func() string { return "abc123" }
			ssMock.OpenStreamFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader(`<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" from="jackal.im" to="jabber.org" version="1.0">`))
				return err
			}
			ssMock.CloseFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("</stream:stream>"))
				return err
			}
			ssMock.SendFunc = func(_ context.Context, element stravaganza.Element) error {
				return element.ToXML(outBuf, true)
			}
			ssMock.ResetFunc = func(tr transport.Transport) error {
				return nil
			}
			trMock.TypeFunc = func() transport.Type { return transport.Socket }
			trMock.StartTLSFunc = func(cfg *tls.Config, asClient bool) {}
			trMock.SetReadRateLimiterFunc = func(rLim *rate.Limiter) error { return nil }
			trMock.CloseFunc = func() error { return nil }

			stm := &outS2S{
				sender: "jackal.im",
				target: "jabber.org",
				cfg: outConfig{
					keepAliveTimeout: time.Minute,
					reqTimeout:       time.Minute,
					maxStanzaSize:    8192,
				},
				typ:     defaultType,
				state:   tt.state,
				flags:   flags{fs: tt.flags},
				rq:      runqueue.New(tt.name),
				tr:      trMock,
				session: ssMock,
				kv:      kvMock,
				hk:      hook.NewHooks(),
				logger:  kitlog.NewNopLogger(),
			}
			// when
			stm.handleSessionResult(tt.sessionResFn())

			// then
			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectedState, stm.getState())

			if tt.expectedFlags > 0 {
				require.Equal(t, tt.expectedFlags, stm.flags.get())
			}
		})
	}
}

func TestDialbackS2S_HandleSessionElement(t *testing.T) {
	var tests = []struct {
		name string

		// input
		state        outState
		sessionResFn func() (stravaganza.Element, error)
		flags        uint8

		// expectations
		expectedOutput        string
		expectedValidDialback bool
		expectedState         outState
	}{
		{
			name:  "Connected/VerifyDialback",
			state: outConnected,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:features").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithChild(
						stravaganza.NewBuilder("dialback").
							WithAttribute(stravaganza.Namespace, dialbackNamespace).
							Build(),
					).
					Build(), nil
			},
			expectedOutput: `<db:verify id="abc123" from="jabber.org" to="jackal.im">1234</db:verify>`,
			expectedState:  outAuthorizingDialbackKey,
		},
		{
			name:  "AuthorizingDialbackKey/Success",
			state: outAuthorizingDialbackKey,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("db:verify").
					WithAttribute(stravaganza.Type, "valid").
					Build(), nil
			},
			expectedOutput:        `</stream:stream>`,
			expectedValidDialback: true,
			expectedState:         outDisconnected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ssMock := &sessionMock{}
			trMock := &transportMock{}

			outBuf := bytes.NewBuffer(nil)
			ssMock.StreamIDFunc = func() string { return "abc123" }
			ssMock.OpenStreamFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader(`<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" from="jackal.im" to="jabber.org" version="1.0">`))
				return err
			}
			ssMock.CloseFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("</stream:stream>"))
				return err
			}
			ssMock.SendFunc = func(_ context.Context, element stravaganza.Element) error {
				return element.ToXML(outBuf, true)
			}
			ssMock.ResetFunc = func(tr transport.Transport) error {
				return nil
			}
			trMock.TypeFunc = func() transport.Type { return transport.Socket }
			trMock.StartTLSFunc = func(cfg *tls.Config, asClient bool) {}
			trMock.SetReadRateLimiterFunc = func(rLim *rate.Limiter) error { return nil }
			trMock.CloseFunc = func() error { return nil }

			stm := &outS2S{
				cfg: outConfig{
					keepAliveTimeout: time.Minute,
					reqTimeout:       time.Minute,
					maxStanzaSize:    8192,
				},
				typ: dialbackType,
				dbParams: DialbackParams{
					StreamID: "abc123",
					From:     "jabber.org",
					To:       "jackal.im",
					Key:      "1234",
				},
				dbResCh: make(chan stream.DialbackResult, 1),
				state:   tt.state,
				flags:   flags{fs: tt.flags},
				rq:      runqueue.New(tt.name),
				tr:      trMock,
				session: ssMock,
				hk:      hook.NewHooks(),
				logger:  kitlog.NewNopLogger(),
			}
			// when
			stm.handleSessionResult(tt.sessionResFn())

			// then
			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectedState, stm.getState())
			if tt.expectedValidDialback {
				time.Sleep(time.Second)
				select {
				case dbRes := <-stm.dbResCh:
					require.True(t, dbRes.Valid)
				default:
					require.Fail(t, "Failed to validate dialback result")
				}
			}
		})
	}
}

func TestOutS2S_HandleSessionError(t *testing.T) {
	var tests = []struct {
		name           string
		state          outState
		sErr           error
		expectedOutput string
		expectClosed   bool
	}{
		{
			name:           "ClosedByPeerError",
			state:          outConnected,
			sErr:           xmppparser.ErrStreamClosedByPeer,
			expectedOutput: `</stream:stream>`,
			expectClosed:   true,
		},
		{
			name:           "EOFError",
			state:          outConnecting,
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

			outBuf := bytes.NewBuffer(nil)
			ssMock.OpenStreamFunc = func(_ context.Context) error {
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

			stm := &outS2S{
				cfg: outConfig{
					keepAliveTimeout: time.Minute,
					reqTimeout:       time.Minute,
					maxStanzaSize:    8192,
				},
				typ:     defaultType,
				state:   tt.state,
				rq:      runqueue.New(tt.name),
				tr:      trMock,
				session: ssMock,
				hk:      hook.NewHooks(),
				logger:  kitlog.NewNopLogger(),
			}
			// when
			stm.handleSessionResult(nil, tt.sErr)

			// then
			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectClosed, trClosed)
		})
	}
}
