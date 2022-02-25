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

package resourcemanager

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/stretchr/testify/require"
)

func TestResourceManager_SetResource(t *testing.T) {
	// given
	var r0, r1 c2smodel.ResourceDesc

	kvmock := &kvMock{}

	h := NewKVManager(kvmock, kitlog.NewNopLogger())
	kvmock.PutFunc = func(ctx context.Context, key string, value string) error {
		r, _ := decodeResource(key, []byte(value))
		r1 = r
		return nil
	}

	// when
	r0 = testResource("megaman-2", 10, "ortuman", "yard")
	err := h.PutResource(context.Background(), r0)

	// then
	require.Nil(t, err)
	require.Equal(t, r0.InstanceID(), r1.InstanceID())
	require.Equal(t, r0.JID().Domain(), r1.JID().Domain())
	require.Equal(t, r0.JID().Resource(), r1.JID().Resource())
	require.Equal(t, r0.Info(), r1.Info())
	require.True(t, reflect.DeepEqual(r0.Presence().String(), r1.Presence().String()))
}

func TestResourceManager_GetResource(t *testing.T) {
	// given
	kvmock := &kvMock{}
	kvmock.PutFunc = func(ctx context.Context, key string, value string) error { return nil }

	h := NewKVManager(kvmock, kitlog.NewNopLogger())

	// when
	r0 := testResource("megaman-2", 10, "ortuman", "yard")
	_ = h.PutResource(context.Background(), r0)

	res, err := h.GetResource(context.Background(), "ortuman", "yard")

	// then
	require.Nil(t, err)
	require.NotNil(t, res)

	require.Equal(t, "ortuman", res.JID().Node())
	require.Equal(t, "jackal.im", res.JID().Domain())
	require.Equal(t, "yard", res.JID().Resource())
	require.Equal(t, true, res.IsAvailable())
	require.Equal(t, int8(10), res.Priority())
	require.Equal(t, "megaman-2", res.InstanceID())
}

func TestResourceManager_GetResources(t *testing.T) {
	// given
	kvmock := &kvMock{}
	kvmock.PutFunc = func(ctx context.Context, key string, value string) error { return nil }

	h := NewKVManager(kvmock, kitlog.NewNopLogger())

	r0 := testResource("abc1234", 100, "ortuman", "yard")
	r1 := testResource("bcd1234", 50, "ortuman", "balcony")
	r2 := testResource("cde1234", 50, "ortuman", "chamber")

	_ = h.PutResource(context.Background(), r0)
	_ = h.PutResource(context.Background(), r1)
	_ = h.PutResource(context.Background(), r2)

	// when
	res, err := h.GetResources(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Len(t, res, 3)
}

func TestResourceManager_DelResource(t *testing.T) {
	// given
	kvmock := &kvMock{}
	kvmock.PutFunc = func(ctx context.Context, key string, value string) error { return nil }

	h := NewKVManager(kvmock, kitlog.NewNopLogger())

	r0 := testResource("megaman-2", 10, "ortuman", "yard")
	_ = h.PutResource(context.Background(), r0)

	var expectedKey string
	kvmock.DelFunc = func(ctx context.Context, key string) error {
		expectedKey = key
		return nil
	}

	// when
	_ = h.DelResource(context.Background(), "ortuman", "yard")

	r1, _ := h.GetResource(context.Background(), "ortuman", "yard")

	// then
	require.Equal(t, fmt.Sprintf("r://ortuman@yard/%s", instance.ID()), expectedKey)

	require.Nil(t, r1)
}

func testResource(instanceID string, priority int8, username, resource string) c2smodel.ResourceDesc {
	pr, _ := stravaganza.NewPresenceBuilder().
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("priority").
				WithText(strconv.Itoa(int(priority))).
				Build(),
		).
		BuildPresence()

	jd, _ := jid.New(username, "jackal.im", resource, true)
	return c2smodel.NewResourceDesc(instanceID, jd, pr, c2smodel.NewInfoMapFromMap(map[string]string{"k1": "v1", "k2": "v2"}))
}
