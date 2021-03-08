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
	"crypto/x509"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackal-xmpp/runqueue"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	xmppparser "github.com/ortuman/jackal/parser"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func init() {
	inDisconnectTimeout = time.Second
}

func TestInS2S_Disconnect(t *testing.T) {
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

	s := &inS2S{
		state:   uint32(inConnected),
		session: sessMock,
		tr:      trMock,
		rq:      runqueue.New("in_s2s:test", nil),
		inHub:   NewInHub(),
		sn:      sonar.New(),
	}
	// when
	s.Disconnect(streamerror.E(streamerror.SystemShutdown))

	time.Sleep(inDisconnectTimeout + time.Second) // wait for disconnect

	// then
	mtx.Lock()
	defer mtx.Unlock()

	require.Equal(t, `<stream:error><system-shutdown xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></stream:error>`, sendBuf.String())
	require.Len(t, sessMock.CloseCalls(), 1)
	require.Len(t, trMock.CloseCalls(), 1)
}

func TestInS2S_HandleSessionElement(t *testing.T) {
	var tests = []struct {
		name string

		// input
		state            inS2SState
		sender           string
		target           string
		sessionResFn     func() (stravaganza.Element, error)
		kvGetFn          func(ctx context.Context, key string) ([]byte, error)
		routeError       error
		flags            uint8
		waitBeforeAssert time.Duration

		// expectations
		expectedOutput string
		expectRouted   bool
		expectedState  inS2SState
		expectedFlags  uint8
	}{
		{
			name:  "Connecting/Unsecured",
			state: inConnecting,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:server").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.To, "localhost").
					WithAttribute(stravaganza.Version, "1.0").
					Build(), nil
			},
			expectedOutput: `<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" id="s2s1" from="localhost" version="1.0"><stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"><starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required/></starttls></stream:features>`,
			expectedState:  inConnected,
		},
		{
			name:  "Connecting/Secured",
			state: inConnecting,
			flags: fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:server").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.To, "localhost").
					WithAttribute(stravaganza.Version, "1.0").
					Build(), nil
			},
			expectedOutput: `<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" id="s2s1" from="localhost" version="1.0"><stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"><mechanisms xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><mechanism>EXTERNAL</mechanism></mechanisms><dialback xmlns="urn:xmpp:features:dialback"/></stream:features>`,
			expectedState:  inConnected,
		},
		{
			name:  "Connecting/SecuredAndAuthenticated",
			state: inConnecting,
			flags: fSecured | fAuthenticated,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:server").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.To, "localhost").
					WithAttribute(stravaganza.Version, "1.0").
					Build(), nil
			},
			expectedOutput: `<?xml version="1.0"?><stream:stream xmlns="jabber:server" xmlns:stream="http://etherx.jabber.org/streams" id="s2s1" from="localhost" version="1.0"><stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"><dialback xmlns="urn:xmpp:features:dialback"/></stream:features>`,
			expectedState:  inConnected,
		},
		{
			name:  "Connected/StartTLS",
			state: inConnected,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("starttls").
					WithAttribute(stravaganza.Namespace, tlsNamespace).
					Build(), nil
			},
			expectedOutput: `<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`,
			expectedState:  inConnecting,
		},
		{
			name:   "Connected/Authenticate",
			state:  inConnected,
			sender: "jabber.org",
			target: "jackal.im",
			flags:  fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("auth").
					WithAttribute(stravaganza.Namespace, saslNamespace).
					WithAttribute("mechanism", "EXTERNAL").
					WithText("=").
					Build(), nil
			},
			expectedOutput: `<success xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`,
			expectedState:  inConnecting,
		},
		{
			name:   "Connected/FailAuthenticate",
			state:  inConnected,
			sender: "konuro.net",
			target: "jackal.im",
			flags:  fSecured,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("auth").
					WithAttribute(stravaganza.Namespace, saslNamespace).
					WithAttribute("mechanism", "EXTERNAL").
					WithText("=").
					Build(), nil
			},
			expectedOutput: `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><bad-protocol/><text>Failed to get peer certificate</text></failure>`,
			expectedState:  inConnected,
		},
		{
			name:   "Connected/VerifyDialbackKey",
			state:  inConnected,
			sender: "jabber.org",
			target: "jackal.im",
			flags:  fSecured | fAuthenticated,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("db:verify").
					WithAttribute(stravaganza.ID, "abc1234").
					WithAttribute(stravaganza.From, "jabber.org").
					WithAttribute(stravaganza.To, "jackal.im").
					WithText("7b909f82401feae55b75289e73d73d0889f1713ae838817feed18bdf427eb03c").
					Build(), nil
			},
			kvGetFn: func(ctx context.Context, key string) ([]byte, error) {
				return []byte("jabber.org jackal.im"), nil
			},
			expectedOutput: `<db:verify from="jackal.im" to="jabber.org" id="abc1234" type="valid"/>`,
			expectedState:  inConnected,
		},
		{
			name:   "Connected/AuthorizeDialbackKey",
			state:  inConnected,
			sender: "jabber.org",
			target: "jackal.im",
			flags:  fSecured | fAuthenticated,
			sessionResFn: func() (stravaganza.Element, error) {
				return stravaganza.NewBuilder("db:result").
					WithAttribute(stravaganza.ID, "abc1234").
					WithAttribute(stravaganza.From, "jabber.org").
					WithAttribute(stravaganza.To, "jackal.im").
					WithText("7b909f82401feae55b75289e73d73d0889f1713ae838817feed18bdf427eb03c").
					Build(), nil
			},
			waitBeforeAssert: time.Second * 2,
			expectedOutput:   `<db:result from="jackal.im" to="jabber.org" type="valid"/>`,
			expectedState:    inConnected,
			expectedFlags:    fSecured | fAuthenticated | fDialbackKeyAuthorized,
		},
		{
			name:  "Connected/RouteIQSuccess",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				iq, _ := stravaganza.NewIQBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.SetType).
					WithAttribute(stravaganza.ID, "iq_1").
					WithChild(
						stravaganza.NewBuilder("ping").
							WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
							Build(),
					).
					BuildIQ(false)
				return iq, nil
			},
			expectedState: inConnected,
			expectRouted:  true,
		},
		{
			name:  "Connected/RouteIQResourceNotFound",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				iq, _ := stravaganza.NewIQBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.SetType).
					WithAttribute(stravaganza.ID, "iq_1").
					WithChild(
						stravaganza.NewBuilder("ping").
							WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
							Build(),
					).
					BuildIQ(false)
				return iq, nil
			},
			routeError:     router.ErrResourceNotFound,
			expectedOutput: `<iq from="noelia@jackal.im/hall" to="ortuman@jabber.org/yard" type="error" id="iq_1"><ping xmlns="urn:xmpp:ping"/><error code="503" type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></error></iq>`,
			expectedState:  inConnected,
		},
		{
			name:  "Connected/RouteIQBlockedSender",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				iq, _ := stravaganza.NewIQBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.SetType).
					WithAttribute(stravaganza.ID, "iq_1").
					WithChild(
						stravaganza.NewBuilder("ping").
							WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
							Build(),
					).
					BuildIQ(false)
				return iq, nil
			},
			routeError:     router.ErrBlockedSender,
			expectedOutput: `<iq from="noelia@jackal.im/hall" to="ortuman@jabber.org/yard" type="error" id="iq_1"><ping xmlns="urn:xmpp:ping"/><error code="503" type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></error></iq>`,
			expectedState:  inConnected,
		},
		{
			name:  "Bounded/RoutePresenceSuccess",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				pr, _ := stravaganza.NewPresenceBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.AvailableType).
					WithAttribute(stravaganza.ID, "pr_1").
					BuildPresence(false)
				return pr, nil
			},
			expectedState: inConnected,
			expectRouted:  true,
		},
		{
			name:  "Bounded/RouteMessageSuccess",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				pr, _ := stravaganza.NewMessageBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.AvailableType).
					WithAttribute(stravaganza.ID, "msg_1").
					WithChild(
						stravaganza.NewBuilder("body").
							WithText("I'll give thee a wind.").
							Build(),
					).
					BuildMessage(false)
				return pr, nil
			},
			expectedState: inConnected,
			expectRouted:  true,
		},
		{
			name:  "Bounded/RouteMessageBlockedSender",
			state: inConnected,
			flags: fSecured | fAuthenticated | fDialbackKeyAuthorized,
			sessionResFn: func() (stravaganza.Element, error) {
				pr, _ := stravaganza.NewMessageBuilder().
					WithAttribute(stravaganza.From, "ortuman@jabber.org/yard").
					WithAttribute(stravaganza.To, "noelia@jackal.im/hall").
					WithAttribute(stravaganza.Type, stravaganza.AvailableType).
					WithAttribute(stravaganza.ID, "msg_1").
					WithChild(
						stravaganza.NewBuilder("body").
							WithText("I'll give thee a wind.").
							Build(),
					).
					BuildMessage(false)
				return pr, nil
			},
			routeError:     router.ErrBlockedSender,
			expectedOutput: `<message from="noelia@jackal.im/hall" to="ortuman@jabber.org/yard" type="error" id="msg_1"><body>I&#39;ll give thee a wind.</body><error code="503" type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></error></message>`,
			expectedState:  inConnected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			trMock := &transportMock{}
			hMock := &hostsMock{}
			kvMock := &kvMock{}
			ssMock := &sessionMock{}
			routerMock := &routerMock{}
			compsMock := &componentsMock{}
			modsMock := &modulesMock{}

			// transport mock
			trMock.PeerCertificatesFunc = func() []*x509.Certificate {
				cert := &x509.Certificate{
					DNSNames: []string{"jabber.org"},
				}
				return []*x509.Certificate{cert}
			}
			trMock.TypeFunc = func() transport.Type { return transport.Socket }
			trMock.StartTLSFunc = func(cfg *tls.Config, asClient bool) {}
			trMock.SetReadRateLimiterFunc = func(rLim *rate.Limiter) error { return nil }
			trMock.CloseFunc = func() error { return nil }

			// hosts mock
			hMock.DefaultHostNameFunc = func() string {
				return "jackal.im"
			}
			hMock.IsLocalHostFunc = func(host string) bool { return host == "jackal.im" }
			hMock.CertificatesFunc = func() []tls.Certificate { return nil }

			// KV mock
			kvMock.GetFunc = tt.kvGetFn
			kvMock.DelFunc = func(ctx context.Context, key string) error { return nil }

			// router mocks
			var routed bool
			routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) error {
				if tt.routeError != nil {
					return tt.routeError
				}
				routed = true
				return nil
			}

			// components mock
			compsMock.IsComponentHostFunc = func(cHost string) bool { return false }

			// modules mock
			modsMock.IsModuleIQFunc = func(iq *stravaganza.IQ) bool { return false }

			// session mock
			outBuf := bytes.NewBuffer(nil)
			ssMock.StreamIDFunc = func() string {
				return "abc123"
			}
			ssMock.OpenStreamFunc = func(ctx context.Context, featuresElem stravaganza.Element) error {
				stmElem := stravaganza.NewBuilder("stream:stream").
					WithAttribute(stravaganza.Namespace, "jabber:server").
					WithAttribute(stravaganza.StreamNamespace, "http://etherx.jabber.org/streams").
					WithAttribute(stravaganza.ID, "s2s1").
					WithAttribute(stravaganza.From, "localhost").
					WithAttribute(stravaganza.Version, "1.0").
					WithChild(featuresElem).
					Build()

				outBuf.WriteString(`<?xml version="1.0"?>`)
				return stmElem.ToXML(outBuf, false)
			}
			ssMock.CloseFunc = func(_ context.Context) error {
				_, err := io.Copy(outBuf, strings.NewReader("</stream:stream>"))
				return err
			}

			var mtx sync.RWMutex
			ssMock.SendFunc = func(_ context.Context, element stravaganza.Element) error {
				mtx.Lock()
				defer mtx.Unlock()
				return element.ToXML(outBuf, true)
			}
			ssMock.SetFromJIDFunc = func(_ *jid.JID) {}
			ssMock.ResetFunc = func(_ transport.Transport) error { return nil }

			// S2S dialback
			dbStreamMock := &s2sDialbackMock{}
			dbStreamMock.DialbackResultFunc = func() <-chan stream.DialbackResult {
				ch := make(chan stream.DialbackResult, 1)
				ch <- stream.DialbackResult{
					Valid: true,
				}
				return ch
			}
			// Out provider mock
			outProviderMock := &outProviderMock{}
			outProviderMock.GetDialbackFunc = func(ctx context.Context, sender string, target string, params DialbackParams) (stream.S2SDialback, error) {
				return dbStreamMock, nil
			}

			stm := &inS2S{
				opts: Options{
					DialbackSecret: "adialbacksecret",
					KeepAlive:      time.Minute,
					RequestTimeout: time.Minute,
					MaxStanzaSize:  8192,
				},
				state:       uint32(tt.state),
				flags:       flags{fs: tt.flags},
				sender:      tt.sender,
				target:      tt.target,
				rq:          runqueue.New(tt.name, nil),
				tr:          trMock,
				hosts:       hMock,
				kv:          kvMock,
				router:      routerMock,
				mods:        modsMock,
				comps:       compsMock,
				session:     ssMock,
				outProvider: outProviderMock,

				sn: sonar.New(),
			}
			// when
			stm.handleSessionResult(tt.sessionResFn())
			if tt.waitBeforeAssert > 0 {
				time.Sleep(tt.waitBeforeAssert)
			}

			// then
			if tt.expectedState == inDisconnected {
				// wait for disconnection
				select {
				case <-stm.Done():
					break
				case <-time.After(inDisconnectTimeout + time.Second):
					break
				}
			}

			mtx.Lock()
			defer mtx.Unlock()

			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectedState, stm.getState())
			require.Equal(t, tt.expectRouted, routed)

			if tt.expectedFlags > 0 {
				require.Equal(t, tt.expectedFlags, stm.flags.get())
			}
		})
	}
}

func TestInS2S_HandleSessionError(t *testing.T) {
	var tests = []struct {
		name           string
		state          inS2SState
		sErr           error
		expectedOutput string
		expectClosed   bool
	}{
		{
			name:           "ClosedByPeerError",
			state:          inConnected,
			sErr:           xmppparser.ErrStreamClosedByPeer,
			expectedOutput: `</stream:stream>`,
			expectClosed:   true,
		},
		{
			name:           "EOFError",
			state:          inConnected,
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
			routerMock := &routerMock{}

			outBuf := bytes.NewBuffer(nil)
			ssMock.OpenStreamFunc = func(_ context.Context, _ stravaganza.Element) error {
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

			stm := &inS2S{
				opts: Options{
					KeepAlive:      time.Minute,
					RequestTimeout: time.Minute,
					MaxStanzaSize:  8192,
				},
				state:   uint32(tt.state),
				rq:      runqueue.New(tt.name, nil),
				tr:      trMock,
				session: ssMock,
				router:  routerMock,
				inHub:   NewInHub(),
				sn:      sonar.New(),
			}
			// when
			stm.handleSessionResult(nil, tt.sErr)

			// then
			require.Equal(t, tt.expectedOutput, outBuf.String())
			require.Equal(t, tt.expectClosed, trClosed)
		})
	}
}
