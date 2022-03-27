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

package session

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/ortuman/jackal/pkg/util/ratelimiter"
	"github.com/stretchr/testify/require"
)

func TestSession_OpenStream(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.TypeFunc = func() transport.Type { return transport.Socket }
	trMock.FlushFunc = func() error { return nil }

	buf := bytes.NewBuffer(nil)
	trMock.WriteStringFunc = func(s string) (int, error) {
		return buf.WriteString(s)
	}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:   C2SSession,
		id:    "ss-1",
		cfg:   Config{MaxStanzaSize: 4096},
		tr:    trMock,
		hosts: &hostsMock{},
		pr:    &xmppParserMock{},
		jd:    *ssJID,
	}

	// when
	err := ss.OpenStream(context.Background())

	// then
	require.Nil(t, err)

	expectedOutput := `<?xml version='1.0'?><stream:stream xmlns='jabber:client' version='1.0' xmlns:stream='http://etherx.jabber.org/streams' from='jackal.im'>`
	require.Equal(t, expectedOutput, buf.String())
}

func TestSession_OpenComponent(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.TypeFunc = func() transport.Type { return transport.Socket }
	trMock.FlushFunc = func() error { return nil }

	buf := bytes.NewBuffer(nil)
	trMock.WriteStringFunc = func(s string) (int, error) {
		return buf.WriteString(s)
	}

	ssJID, _ := jid.NewWithString("upload.jackal.im", true)
	ss := Session{
		typ:      ComponentSession,
		id:       "comp-1",
		streamID: "stm-1",
		cfg:      Config{MaxStanzaSize: 4096},
		tr:       trMock,
		hosts:    &hostsMock{},
		pr:       &xmppParserMock{},
		jd:       *ssJID,
	}

	// when
	err := ss.OpenComponent(context.Background())

	// then
	require.Nil(t, err)

	expectedOutput := `<?xml version='1.0'?><stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='upload.jackal.im' id='stm-1'>`
	require.Equal(t, expectedOutput, buf.String())
}

func TestSession_OpenServer(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.TypeFunc = func() transport.Type { return transport.Socket }
	trMock.FlushFunc = func() error { return nil }

	buf := bytes.NewBuffer(nil)
	trMock.WriteStringFunc = func(s string) (int, error) {
		return buf.WriteString(s)
	}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:   S2SSession,
		id:    "ss-1",
		cfg:   Config{MaxStanzaSize: 4096},
		tr:    trMock,
		hosts: &hostsMock{},
		pr:    &xmppParserMock{},
		jd:    *ssJID,
	}

	// when
	err := ss.OpenStream(context.Background())

	// then
	require.Nil(t, err)

	expectedOutput := `<?xml version='1.0'?><stream:stream xmlns='jabber:server' version='1.0' xmlns:stream='http://etherx.jabber.org/streams' xmlns:db='jabber:server:dialback' from='jackal.im'>`
	require.Equal(t, expectedOutput, buf.String())
}

func TestSession_Close(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.TypeFunc = func() transport.Type { return transport.Socket }
	trMock.FlushFunc = func() error { return nil }

	buf := bytes.NewBuffer(nil)
	trMock.WriteStringFunc = func(s string) (int, error) {
		return buf.WriteString(s)
	}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:    C2SSession,
		id:     "ss-1",
		cfg:    Config{MaxStanzaSize: 4096},
		tr:     trMock,
		hosts:  &hostsMock{},
		pr:     &xmppParserMock{},
		jd:     *ssJID,
		opened: true,
	}

	// when
	err := ss.Close(context.Background())

	// then
	require.Nil(t, err)

	expectedOutput := `</stream:stream>`
	require.Equal(t, expectedOutput, buf.String())
}

func TestSession_Send(t *testing.T) {
	// given
	trMock := &transportMock{}
	trMock.TypeFunc = func() transport.Type { return transport.Socket }
	trMock.FlushFunc = func() error { return nil }

	buf := bytes.NewBuffer(nil)
	trMock.WriteStringFunc = func(s string) (int, error) {
		return buf.WriteString(s)
	}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:    C2SSession,
		id:     "ss-1",
		cfg:    Config{MaxStanzaSize: 4096},
		tr:     trMock,
		hosts:  &hostsMock{},
		pr:     &xmppParserMock{},
		jd:     *ssJID,
		opened: true,
	}

	// when
	err := ss.Send(context.Background(), stravaganza.NewBuilder("foo-stanza").Build())

	// then
	require.Nil(t, err)

	expectedOutput := `<foo-stanza/>`
	require.Equal(t, expectedOutput, buf.String())
}

func TestSession_ReceiveStreamSuccess(t *testing.T) {
	// given
	hMock := &hostsMock{}
	trMock := &transportMock{}
	prMock := &xmppParserMock{}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:     C2SSession,
		id:      "ss-1",
		cfg:     Config{MaxStanzaSize: 4096},
		tr:      trMock,
		hosts:   hMock,
		pr:      prMock,
		jd:      *ssJID,
		opened:  true,
		started: false,
	}
	hMock.IsLocalHostFunc = func(domain string) bool { return domain == "jackal.im" }
	trMock.TypeFunc = func() transport.Type { return transport.Socket }

	prMock.ParseFunc = func() (stravaganza.Element, error) {
		return stravaganza.NewBuilder("stream:stream").
			WithAttribute(stravaganza.Namespace, jabberClientNamespace).
			WithAttribute("xmlns:stream", streamNamespace).
			WithAttribute(stravaganza.To, "jackal.im").
			WithAttribute(stravaganza.Version, "1.0").
			Build(), nil
	}

	// when
	elem, err := ss.Receive()

	// then
	require.Nil(t, err)
	require.NotNil(t, elem)

	require.Equal(t, "stream:stream", elem.Name())
}

func TestSession_ReceiveBadStream(t *testing.T) {
	// given
	hMock := &hostsMock{}
	trMock := &transportMock{}
	prMock := &xmppParserMock{}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:     C2SSession,
		id:      "ss-1",
		cfg:     Config{MaxStanzaSize: 4096},
		tr:      trMock,
		hosts:   hMock,
		pr:      prMock,
		jd:      *ssJID,
		opened:  true,
		started: false,
	}
	hMock.IsLocalHostFunc = func(domain string) bool { return domain == "jackal.im" }
	trMock.TypeFunc = func() transport.Type { return transport.Socket }

	prMock.ParseFunc = func() (stravaganza.Element, error) {
		return stravaganza.NewBuilder("stream:stream").
			WithAttribute(stravaganza.Namespace, jabberClientNamespace).
			WithAttribute("xmlns:stream", streamNamespace).
			WithAttribute(stravaganza.To, "jackal.im").
			WithAttribute(stravaganza.Version, "2.0"). // invalid version
			Build(), nil
	}

	// when
	_, err := ss.Receive()

	// then
	require.NotNil(t, err)

	se, ok := err.(*streamerror.Error)
	require.True(t, ok)
	require.Equal(t, streamerror.UnsupportedVersion, se.Reason)
}

func TestSession_ReceiveSuccess(t *testing.T) {
	// given
	prMock := &xmppParserMock{}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:     C2SSession,
		id:      "ss-1",
		cfg:     Config{MaxStanzaSize: 4096},
		tr:      &transportMock{},
		hosts:   &hostsMock{},
		pr:      prMock,
		jd:      *ssJID,
		opened:  true,
		started: true,
	}

	// when
	prMock.ParseFunc = func() (stravaganza.Element, error) {
		b := stravaganza.NewMessageBuilder()
		b.WithAttribute("from", "noelia@jackal.im/yard")
		b.WithAttribute("to", "ortuman@jackal.im/balcony")
		b.WithChild(
			stravaganza.NewBuilder("body").
				WithText("I'll give thee a wind.").
				Build(),
		)
		msg, _ := b.BuildMessage()
		return msg, nil
	}
	elem, err := ss.Receive()

	// then
	require.Nil(t, err)
	require.NotNil(t, elem)

	require.Equal(t, "message", elem.Name())
}

func TestSession_ReceiveStreamError(t *testing.T) {
	// given
	prMock := &xmppParserMock{}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:     C2SSession,
		id:      "ss-1",
		cfg:     Config{MaxStanzaSize: 4096},
		tr:      &transportMock{},
		hosts:   &hostsMock{},
		pr:      prMock,
		jd:      *ssJID,
		opened:  true,
		started: true,
	}

	// when
	errFoo := errors.New("foo error")
	prMock.ParseFunc = func() (stravaganza.Element, error) { return nil, errFoo } // unmapped error
	_, err0 := ss.Receive()

	prMock.ParseFunc = func() (stravaganza.Element, error) { return nil, ratelimiter.ErrReadLimitExcedeed }
	_, err1 := ss.Receive()

	prMock.ParseFunc = func() (stravaganza.Element, error) { return nil, xmppparser.ErrTooLargeStanza }
	_, err2 := ss.Receive()

	prMock.ParseFunc = func() (stravaganza.Element, error) { return nil, &xml.SyntaxError{} }
	_, err3 := ss.Receive()

	// then
	require.NotNil(t, err0)
	require.NotNil(t, err1)
	require.NotNil(t, err2)
	require.NotNil(t, err3)

	require.Equal(t, errFoo, err0)

	se1, ok1 := err1.(*streamerror.Error)
	se2, ok2 := err2.(*streamerror.Error)
	se3, ok3 := err3.(*streamerror.Error)
	require.True(t, ok1)
	require.True(t, ok2)
	require.True(t, ok3)

	require.Equal(t, streamerror.PolicyViolation, se1.Reason)
	require.Equal(t, streamerror.PolicyViolation, se2.Reason)
	require.Equal(t, streamerror.InvalidXML, se3.Reason)
}

func TestSession_ReceiveUnsupportedStanza(t *testing.T) {
	// given
	prMock := &xmppParserMock{}

	ssJID, _ := jid.NewWithString("jackal.im", true)
	ss := Session{
		typ:     C2SSession,
		id:      "ss-1",
		cfg:     Config{MaxStanzaSize: 4096},
		tr:      &transportMock{},
		hosts:   &hostsMock{},
		pr:      prMock,
		jd:      *ssJID,
		opened:  true,
		started: true,
	}

	// when
	prMock.ParseFunc = func() (stravaganza.Element, error) {
		return stravaganza.NewBuilder("iq").Build(), nil
	}
	_, err := ss.Receive()

	// then
	require.NotNil(t, err)

	se, ok := err.(*stanzaerror.Error)

	require.True(t, ok)
	require.Equal(t, stanzaerror.BadRequest, se.Reason)
}
