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
	"testing"

	kitlog "github.com/go-kit/log"

	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/stretchr/testify/require"
)

func TestInHub_StartStop(t *testing.T) {
	// given
	mockStm := &s2sInMock{}
	mockStm.IDFunc = func() stream.S2SInID { return 1234 }
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

	h := &InHub{
		streams: make(map[stream.S2SInID]stream.S2SIn),
		doneCh:  make(chan chan struct{}),
		logger:  kitlog.NewNopLogger(),
	}

	// when
	_ = h.Start(context.Background())

	h.register(mockStm)

	_ = h.Stop(context.Background())

	// then
	require.Len(t, mockStm.DisconnectCalls(), 1)
	require.Equal(t, discReason, streamerror.SystemShutdown)
}
