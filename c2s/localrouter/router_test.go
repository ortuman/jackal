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

package localrouter

import (
	"context"
	"sync"
	"testing"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/router/stream"
	"github.com/stretchr/testify/require"
)

func TestRouter_RegisterBind(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
	}

	// when
	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	// then
	require.Len(t, r.stms, 0)
	require.Len(t, r.bndRes, 1)

	require.NotNil(t, r.bndRes["ortuman"])
}

func TestRouter_Stream(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
	}

	// when
	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	stm := r.Stream("ortuman", "yard")

	// then
	require.NotNil(t, stm)
	require.Equal(t, mockStm, stm)
}

func TestRouter_Stop(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }
	mockStm.DoneFunc = func() <-chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	}

	var discReason streamerror.Reason
	mockStm.DisconnectFunc = func(streamErr *streamerror.Error) <-chan error {
		discReason = streamErr.Reason
		return nil
	}

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
		doneCh: make(chan chan struct{}),
	}

	// when
	_ = r.Start(context.Background())

	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	_ = r.Stop(context.Background())

	// then
	require.Len(t, mockStm.DisconnectCalls(), 1)
	require.Equal(t, discReason, streamerror.SystemShutdown)
}

func TestRouter_Unregister(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
	}

	// when
	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	_ = r.Unregister(mockStm)

	// then
	require.Len(t, r.stms, 0)
	require.Len(t, r.bndRes, 0)

	require.Nil(t, r.bndRes["ortuman"])
}

func TestRouter_Route(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }

	var mu sync.RWMutex
	var sentElement stravaganza.Element

	mockStm.SendElementFunc = func(elem stravaganza.Element) <-chan error {
		mu.Lock()
		sentElement = elem
		mu.Unlock()
		return nil
	}

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
	}

	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	// when
	stanza := testMessageStanza()
	err := r.Route(stanza, "ortuman", "yard")

	// then
	mu.Lock()
	defer mu.Unlock()

	require.Nil(t, err)
	require.Equal(t, stanza.String(), sentElement.String())
}

func TestRouter_Disconnect(t *testing.T) {
	// given
	mockStm := &streamC2SMock{}
	mockStm.IDFunc = func() stream.C2SID { return 1234 }
	mockStm.UsernameFunc = func() string { return "ortuman" }
	mockStm.ResourceFunc = func() string { return "yard" }

	mockStm.DisconnectFunc = func(streamErr *streamerror.Error) <-chan error {
		errCh := make(chan error, 1)
		errCh <- nil
		return errCh
	}

	r := &Router{
		hosts:  &hostsMock{},
		sonar:  sonar.New(),
		stms:   make(map[stream.C2SID]stream.C2S),
		bndRes: make(map[string]*resources),
	}

	_ = r.Register(mockStm)
	_, _ = r.Bind(1234)

	// when
	err := r.Disconnect("ortuman", "yard", streamerror.E(streamerror.SystemShutdown))

	require.Nil(t, err)
	require.Len(t, mockStm.DisconnectCalls(), 1)
}

func testMessageStanza() *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/hall")
	b.WithAttribute("to", "ortuman@jackal.im/yard")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage(true)
	return msg
}
