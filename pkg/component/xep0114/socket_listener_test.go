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
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSocketListener_Listen(t *testing.T) {
	// given
	var handledConn uint32

	s := &SocketListener{
		cfg:    ListenerConfig{Port: 51126},
		stmHub: newInHub(),
		connHandlerFn: func(_ net.Conn) {
			atomic.StoreUint32(&handledConn, 1)
		},
	}

	// when
	err := s.Start(context.Background())
	require.Nil(t, err)

	_, err = net.Dial("tcp", ":51126")
	require.Nil(t, err)

	time.Sleep(time.Second) // wait to accept

	handled := atomic.LoadUint32(&handledConn) == 1
	_ = s.Stop(context.Background())

	// then
	require.True(t, handled)

	require.Equal(t, uint32(0), atomic.LoadUint32(&s.active))
}
