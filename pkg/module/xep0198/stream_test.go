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
	"context"
	"testing"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/hook"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/stretchr/testify/require"
)

func TestStream_EncodeSMID(t *testing.T) {
	// given
	jd, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	nonce := make([]byte, nonceLength)
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	// when
	smID := encodeSMID(jd, nonce)

	// then
	require.Equal(t, "b3J0dW1hbkBqYWNrYWwuaW0veWFyZAABAgMEBQYHCAkKCwwNDg8QERITFBUWFxg=", smID)
}

func TestStream_DecodeSMID(t *testing.T) {
	// given
	smID := "b3J0dW1hbkBqYWNrYWwuaW0vQ29udmVyc2F0aW9ucy40UllFAHl5Jrx+gnSZ7hq3vjoW38oQM2ZrPknCyA=="

	// when
	jd, nonce, err := decodeSMID(smID)

	// then
	require.Nil(t, err)
	require.NotNil(t, jd)
	require.NotNil(t, nonce)

	expectedNonce := []byte{
		0x79, 0x79, 0x26, 0xbc, 0x7e, 0x82, 0x74, 0x99,
		0xee, 0x1a, 0xb7, 0xbe, 0x3a, 0x16, 0xdf, 0xca,
		0x10, 0x33, 0x66, 0x6b, 0x3e, 0x49, 0xc2, 0xc8,
	}
	require.Equal(t, "ortuman@jackal.im/Conversations.4RYE", jd.String())
	require.Equal(t, expectedNonce, nonce)
}

func TestStream_Enable(t *testing.T) {
	// given
	jd, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	stmMock := &c2sStreamMock{}

	var setK string
	var setVal interface{}
	stmMock.IDFunc = func() stream.C2SID { return 1234 }
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		setK = k
		setVal = val
		return nil
	}
	stmMock.IsBindedFunc = func() bool { return true }
	stmMock.JIDFunc = func() *jid.JID { return jd }
	stmMock.UsernameFunc = func() string { return jd.Node() }
	stmMock.ResourceFunc = func() string { return jd.Resource() }
	stmMock.InfoFunc = func() c2smodel.Info { return c2smodel.Info{M: map[string]string{}} }

	var sentEl stravaganza.Element
	stmMock.SendElementFunc = func(elem stravaganza.Element) <-chan error {
		sentEl = elem
		return nil
	}

	hk := hook.NewHooks()
	sm := &Stream{
		cfg:    testSMConfig(),
		queues: make(map[string]*queue),
		hk:     hk,
	}

	// when
	_ = sm.Start(context.Background())
	defer func() { _ = sm.Stop(context.Background()) }()

	halted, err := hk.Run(context.Background(), hook.C2SStreamElementReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{
			Element: stravaganza.NewBuilder("enable").
				WithAttribute(stravaganza.Namespace, streamNamespace).
				Build(),
		},
		Sender: stmMock,
	})

	// then
	require.True(t, halted)
	require.Nil(t, err)

	require.Equal(t, setK, enabledInfoKey)
	require.Equal(t, true, setVal)

	require.Equal(t, "enabled", sentEl.Name())
	require.Equal(t, streamNamespace, sentEl.Attribute(stravaganza.Namespace))

	sq := sm.queues[queueKey(jd)]
	require.NotNil(t, sq)
}

func TestStream_InStanza(t *testing.T) {
	// given
	// when
	// then
}

func TestStream_OutStanza(t *testing.T) {
	// given
	// when
	// then
}

func TestStream_OutStanzaMaxQueueSizeReached(t *testing.T) {
	// given
	// when
	// then
}

func TestStream_R(t *testing.T) {
	// given
	// when
	// then
}

func TestStream_Acknowledge(t *testing.T) {
	// given
	// when
	// then
}

func TestStream_Resume(t *testing.T) {
	// given
	// when
	// then
}

func testSMConfig() Config {
	return Config{
		HibernateTime:      time.Second,
		RequestAckInterval: time.Second,
		WaitForAckTimeout:  time.Second,
		MaxQueueSize:       10,
	}
}
