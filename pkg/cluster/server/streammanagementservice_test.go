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

package clusterserver

import (
	"context"
	"math/rand"
	"testing"
	"time"

	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/pb"
	streamqueue "github.com/ortuman/jackal/pkg/module/xep0198/queue"
	"github.com/stretchr/testify/require"
)

func TestStreamManagementService_TransferQueue(t *testing.T) {
	// given
	stmMock := &c2sStreamMock{}
	stmMock.DisconnectFunc = func(streamErr *streamerror.Error) <-chan error {
		errCh := make(chan error, 1)
		errCh <- nil
		return errCh
	}

	elements := []streamqueue.Element{
		{
			Stanza: testMessageStanza(),
			H:      10,
		},
	}
	nonce := make([]byte, 16)
	for i := range nonce {
		nonce[i] = byte(rand.Intn(255) + 1)
	}

	q := streamqueue.New(
		stmMock,
		nonce,
		elements,
		5,
		10,
		time.Second*5,
		time.Second*5,
	)

	sm := streamqueue.NewQueueMap()
	sm.Set("q1", q)

	srv := &streamManagementService{stmQueueMap: sm}

	// when
	resp, err := srv.TransferQueue(context.Background(), &pb.TransferQueueRequest{
		Identifier: "q1",
	})
	require.NoError(t, err)

	// then
	require.Len(t, stmMock.DisconnectCalls(), 1)

	require.Len(t, resp.Elements, 1)
	require.Equal(t, nonce, resp.Nonce)
	require.Equal(t, uint32(5), resp.InH)
	require.Equal(t, uint32(10), resp.OutH)
}
