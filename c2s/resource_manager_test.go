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

package c2s

import (
	"context"
	"reflect"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestResourceManager_SetResource(t *testing.T) {
	// given
	var r0, r1 model.Resource
	r0 = testResource("megaman-2", 10)

	kvmock := &kvMock{}

	h := &ResourceManager{kv: kvmock}
	kvmock.PutFunc = func(ctx context.Context, key string, value string) error {
		r, _ := decodeResource(key, []byte(value))
		r1 = *r
		return nil
	}

	// when
	err := h.PutResource(context.Background(), &r0)

	// then
	require.Nil(t, err)
	require.Equal(t, r0.InstanceID, r1.InstanceID)
	require.Equal(t, r0.JID.Domain(), r1.JID.Domain())
	require.Equal(t, r0.JID.Resource(), r1.JID.Resource())
	require.Equal(t, r0.Context, r1.Context)
	require.True(t, reflect.DeepEqual(r0.Presence.String(), r1.Presence.String()))
}

func TestResourceManager_GetResource(t *testing.T) {
	// given
	kvmock := &kvMock{}

	h := &ResourceManager{kv: kvmock}

	var expectedKey string
	kvmock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		expectedKey = prefix

		res := testResource("inst-1", 100)
		b, _ := resourceVal(&res)

		return map[string][]byte{"r://ortuman@balcony": b}, nil
	}

	// when
	res, err := h.GetResource(context.Background(), "ortuman", "balcony")

	// then
	require.Nil(t, err)

	require.Equal(t, "r://ortuman@balcony", expectedKey)

	require.Equal(t, "ortuman", res.JID.Node())
	require.Equal(t, "jackal.im", res.JID.Domain())
	require.Equal(t, "balcony", res.JID.Resource())
	require.Equal(t, true, res.IsAvailable())
	require.Equal(t, int8(100), res.Priority())
	require.Equal(t, "inst-1", res.InstanceID)
}

func TestResourceManager_GetResources(t *testing.T) {
	// given
	kvmock := &kvMock{}

	h := &ResourceManager{kv: kvmock}

	var expectedKey string
	kvmock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		expectedKey = prefix

		r0 := testResource("abc1234", 100)
		r1 := testResource("bcd1234", 50)
		r2 := testResource("cde1234", 50)

		b0, _ := resourceVal(&r0)
		b1, _ := resourceVal(&r1)
		b2, _ := resourceVal(&r2)

		return map[string][]byte{
			"r://ortuman@balcony": b0,
			"r://ortuman@yard":    b1,
			"r://ortuman@hall":    b2,
		}, nil
	}

	// when
	res, err := h.GetResources(context.Background(), "ortuman")

	// then
	require.Nil(t, err)

	require.Equal(t, "r://ortuman", expectedKey)
	require.Len(t, res, 3)
}

func TestResourceManager_DelResource(t *testing.T) {
	// given
	kvmock := &kvMock{}

	h := &ResourceManager{kv: kvmock}

	var expectedKey string
	kvmock.DelFunc = func(ctx context.Context, key string) error {
		expectedKey = key
		return nil
	}

	// when
	_ = h.DelResource(context.Background(), "ortuman", "yard")

	// then
	require.Equal(t, "r://ortuman@yard", expectedKey)
}
