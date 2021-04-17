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
	"context"
	"errors"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
	"github.com/stretchr/testify/require"
)

func TestS2sRouter_RouteSuccess(t *testing.T) {
	// given
	out := &s2sOutMock{}
	out.SendElementFunc = func(elem stravaganza.Element) <-chan error {
		return nil
	}
	op := &outProviderMock{}
	op.GetOutFunc = func(ctx context.Context, sender string, target string) (stream.S2SOut, error) {
		return out, nil
	}

	// when
	r := &s2sRouter{
		outProvider: op,
	}
	err := r.Route(context.Background(), testMessageStanza(), "jackal.im")

	// then
	require.Nil(t, err)
	require.Len(t, out.SendElementCalls(), 1)
}

func TestS2sRouter_RouteServerTimeoutError(t *testing.T) {
	// given
	op := &outProviderMock{}
	op.GetOutFunc = func(ctx context.Context, sender string, target string) (stream.S2SOut, error) {
		return nil, errServerTimeout
	}

	// when
	r := &s2sRouter{
		outProvider: op,
	}
	err := r.Route(context.Background(), testMessageStanza(), "jackal.im")

	// then
	require.Equal(t, router.ErrRemoteServerTimeout, err)
}

func TestS2sRouter_RouteServerGenericError(t *testing.T) {
	// given
	op := &outProviderMock{}
	op.GetOutFunc = func(ctx context.Context, sender string, target string) (stream.S2SOut, error) {
		return nil, errors.New("foo error")
	}

	// when
	r := &s2sRouter{
		outProvider: op,
	}
	err := r.Route(context.Background(), testMessageStanza(), "jackal.im")

	// then
	require.Equal(t, router.ErrRemoteServerNotFound, err)
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
	msg, _ := b.BuildMessage(true)
	return msg
}
