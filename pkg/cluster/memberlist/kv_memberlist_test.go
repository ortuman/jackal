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
	"fmt"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	kvtypes "github.com/ortuman/jackal/pkg/cluster/kv/types"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/stretchr/testify/require"
)

func TestMemberList_Join(t *testing.T) {
	// given
	kvMock := &kvMock{}

	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kvtypes.WatchResp {
		return make(chan kvtypes.WatchResp)
	}
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		return nil
	}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return map[string][]byte{
			fmt.Sprintf("i://%s", instance.ID()): []byte(fmt.Sprintf("a=%s:4312 cv=v1.0.0", instance.Hostname())),
			"i://b3fd":                           []byte("a=192.168.0.12:1456 cv=v1.0.0"),
		}, nil
	}
	ml := NewKVMemberList(4312, kvMock, hook.NewHooks(), kitlog.NewNopLogger())

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

	wCh := make(chan kvtypes.WatchResp)
	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kvtypes.WatchResp {
		return wCh
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
	ml := NewKVMemberList(4312, kvMock, hook.NewHooks(), kitlog.NewNopLogger())

	// when
	_ = ml.Start(context.Background())

	close(wCh)
	err := ml.Stop(context.Background())

	// then
	require.Nil(t, err)

	require.Len(t, kvMock.DelCalls(), 1)
}

func TestMemberList_WatchChanges(t *testing.T) {
	// given
	kvMock := &kvMock{}

	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan kvtypes.WatchResp {
		wCh := make(chan kvtypes.WatchResp)
		go func() {
			wCh <- kvtypes.WatchResp{
				Events: []kvtypes.WatchEvent{
					{Type: kvtypes.Del, Key: "b3fd"},
					{Type: kvtypes.Put, Key: "c5gl", Val: []byte("a=192.168.0.14:4256 cv=v1.5.0")},
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
			fmt.Sprintf("i://%s", instance.ID()): []byte(fmt.Sprintf("a=%s:4312 cv=v1.0.0", instance.Hostname())),
			"i://b3fd":                           []byte("a=192.168.0.12:1456 cv=v1.0.0"),
		}, nil
	}
	ml := NewKVMemberList(4312, kvMock, hook.NewHooks(), kitlog.NewNopLogger())

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
