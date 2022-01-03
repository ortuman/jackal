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

package extcomponentmanager

import (
	"context"
	"sync"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"

	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/stretchr/testify/require"
)

func TestManager_RegisterComponentHost(t *testing.T) {
	// given
	kvMock := &kvMock{}

	var k, v string
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		k = key
		v = value
		return nil
	}
	m := &Manager{
		kv: kvMock,
	}

	// when
	_ = m.RegisterComponentHost(context.Background(), "muc.jackal.im")

	// then
	require.Len(t, kvMock.PutCalls(), 1)

	require.Equal(t, "ec://muc.jackal.im", k)
	require.Equal(t, "i="+instance.ID(), v)
}

func TestManager_UnregisterComponentHost(t *testing.T) {
	// given
	kvMock := &kvMock{}

	var k string
	kvMock.DelFunc = func(ctx context.Context, key string) error {
		k = key
		return nil
	}
	m := &Manager{
		kv: kvMock,
	}

	// when
	_ = m.UnregisterComponentHost(context.Background(), "muc.jackal.im")

	// then
	require.Len(t, kvMock.DelCalls(), 1)

	require.Equal(t, "ec://muc.jackal.im", k)
}

func TestManager_RegisterExternalComponent(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{
			"ec://muc.jackal.im": []byte("i=inst-1"),
		}, nil
	}
	wCh := make(chan kv.WatchResp)
	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
		return wCh
	}

	clConnMngMock := &clusterConnManagerMock{}
	clConnMngMock.GetConnectionFunc = func(instanceID string) (clusterconnmanager.Conn, error) {
		return &clusterConnMock{}, nil
	}

	var mu sync.RWMutex
	var registeredHosts []string

	compsMock := &componentsMock{}
	compsMock.RegisterComponentFunc = func(_ context.Context, comp component.Component) error {
		mu.Lock()
		defer mu.Unlock()
		registeredHosts = append(registeredHosts, comp.Host())
		return nil
	}

	m := &Manager{
		kv:             kvMock,
		clusterConnMng: clConnMngMock,
		comps:          compsMock,
		logger:         kitlog.NewNopLogger(),
	}
	// when
	_ = m.Start(context.Background())

	time.Sleep(time.Millisecond * 250)

	wCh <- kv.WatchResp{
		Events: []kv.WatchEvent{
			{Type: kv.Put, Key: "ec://pubsub.jackal.im", Val: []byte("i=inst-2")},
		},
	}

	time.Sleep(time.Millisecond * 250)

	// then
	mu.RLock()
	defer mu.RUnlock()

	require.Len(t, compsMock.RegisterComponentCalls(), 2)
	require.Equal(t, []string{"muc.jackal.im", "pubsub.jackal.im"}, registeredHosts)
}

func TestManager_UnregisterExternalComponent(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{
			"ec://muc.jackal.im":    []byte("i=inst-1"),
			"ec://pubsub.jackal.im": []byte("i=inst-2"),
		}, nil
	}
	wCh := make(chan kv.WatchResp)
	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
		return wCh
	}

	clConnMngMock := &clusterConnManagerMock{}
	clConnMngMock.GetConnectionFunc = func(instanceID string) (clusterconnmanager.Conn, error) {
		return &clusterConnMock{}, nil
	}

	var mu sync.RWMutex
	var unregisteredHost string

	compsMock := &componentsMock{}
	compsMock.RegisterComponentFunc = func(_ context.Context, comp component.Component) error {
		return nil
	}
	compsMock.UnregisterComponentFunc = func(_ context.Context, cHost string) error {
		mu.Lock()
		defer mu.Unlock()
		unregisteredHost = cHost
		return nil
	}

	m := &Manager{
		kv:             kvMock,
		clusterConnMng: clConnMngMock,
		comps:          compsMock,
		logger:         kitlog.NewNopLogger(),
	}
	// when
	_ = m.Start(context.Background())

	time.Sleep(time.Millisecond * 250)

	wCh <- kv.WatchResp{
		Events: []kv.WatchEvent{
			{Type: kv.Del, Key: "ec://pubsub.jackal.im"},
		},
	}
	time.Sleep(time.Millisecond * 250)

	// then
	mu.RLock()
	defer mu.RUnlock()

	require.Len(t, compsMock.RegisterComponentCalls(), 2)
	require.Len(t, compsMock.UnregisterComponentCalls(), 1)

	require.Equal(t, "pubsub.jackal.im", unregisteredHost)
}
