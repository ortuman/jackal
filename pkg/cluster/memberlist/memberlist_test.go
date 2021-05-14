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

package memberlist

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ortuman/jackal/pkg/module/hook"

	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = os.Setenv("JACKAL_INSTANCE_ID", "af2d")

	interfaceAddrs = func() ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{
			IP:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 0, 13},
			Mask: []byte{255, 255, 255, 0},
		}}, nil
	}
}

func TestMemberList_Join(t *testing.T) {
	// given
	kvMock := &kvMock{}

	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
		return make(chan kv.WatchResp)
	}
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		return nil
	}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{
			"i://af2d": []byte("a=192.168.0.13:4312 cv=v1.0.0"),
			"i://b3fd": []byte("a=192.168.0.12:1456 cv=v1.0.0"),
		}, nil
	}
	ml := New(kvMock, 4312, hook.NewHooks())

	// when
	err := ml.Start(context.Background())

	m, ok := ml.GetMember("b3fd")

	ms := ml.GetMembers()

	// then
	require.Nil(t, err)

	require.True(t, ok)
	require.Equal(t, "192.168.0.12", m.Host)
	require.Equal(t, 1456, m.Port)
	require.Len(t, ms, 1)
}

func TestMemberList_Leave(t *testing.T) {
	// given
	kvMock := &kvMock{}

	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
		return make(chan kv.WatchResp)
	}
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		return nil
	}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{"i://af2d": []byte("a=192.168.0.13:4312 cv=v1.0.0")}, nil
	}
	kvMock.DelFunc = func(r context.Context, key string) error {
		return nil
	}
	ml := New(kvMock, 4312, hook.NewHooks())

	// when
	_ = ml.Start(context.Background())

	err := ml.Stop(context.Background())

	// then
	require.Nil(t, err)

	require.Len(t, kvMock.DelCalls(), 1)
}

func TestMemberList_WatchChanges(t *testing.T) {
	// given
	kvMock := &kvMock{}

	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kv.WatchResp {
		wCh := make(chan kv.WatchResp)
		go func() {
			wCh <- kv.WatchResp{
				Events: []kv.WatchEvent{
					{Type: kv.Del, Key: "b3fd"},
					{Type: kv.Put, Key: "c5gl", Val: []byte("a=192.168.0.14:4256 cv=v1.5.0")},
				},
			}
		}()
		return wCh
	}
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		return nil
	}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{
			"i://af2d": []byte("a=192.168.0.13:4312 cv=v1.0.0"),
			"i://b3fd": []byte("a=192.168.0.12:1456 cv=v1.0.0"),
		}, nil
	}
	ml := New(kvMock, 4312, hook.NewHooks())

	// when
	_ = ml.Start(context.Background())

	time.Sleep(time.Millisecond * 1500)

	ms := ml.GetMembers()

	// then
	require.Len(t, ms, 1)

	_, ok := ms["b3fd"]
	require.False(t, ok) // deleted key

	// updated/registered keys
	_, ok = ms["af2d"]
	require.False(t, ok)

	_, ok = ms["c5gl"]
	require.True(t, ok)
}
