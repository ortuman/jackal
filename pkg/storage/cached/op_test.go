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

package cachedrepository

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-kit/log"

	"github.com/golang/protobuf/proto"
	"github.com/ortuman/jackal/pkg/model"
	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/stretchr/testify/require"
)

func TestCachedRepository_ExistsOp(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.HasKeyFunc = func(ctx context.Context, ns, k string) (bool, error) {
		if ns == "n0" && k == "k0" {
			return true, nil
		}
		return false, nil
	}
	missFn := func(context.Context) (bool, error) {
		return false, nil
	}

	// when
	op0 := existsOp{c: cacheMock, namespace: "n0", key: "k0", missFn: missFn}
	op1 := existsOp{c: cacheMock, namespace: "n1", key: "k1", missFn: missFn}

	ok0, _ := op0.do(context.Background())
	ok1, _ := op1.do(context.Background())

	// then
	require.True(t, ok0)
	require.False(t, ok1)
}

func TestCachedRepository_UpdateOp(t *testing.T) {
	// given
	var output string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		output += "del"
		return nil
	}
	updateFn := func(context.Context) error {
		output += ":rep_update"
		return nil
	}

	// when
	op := updateOp{c: cacheMock, invalidateKeys: []string{"k0"}, updateFn: updateFn}
	_ = op.do(context.Background())

	// then
	require.Equal(t, "del:rep_update", output) // ensure proper order
}

func TestCachedRepository_FetchOpHit(t *testing.T) {
	// given
	var usr usermodel.User
	usr.Username = "ortuman"

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return proto.Marshal(&usr)
	}

	var missed bool
	op := fetchOp{
		c:     cacheMock,
		codec: &usr,
		missFn: func(ctx context.Context) (model.Codec, error) {
			missed = true
			return nil, nil
		},
		logger: log.NewNopLogger(),
	}

	// when
	v, _ := op.do(context.Background())

	// then
	require.False(t, missed)
	require.Equal(t, "ortuman", v.(*usermodel.User).Username)
}

func TestCachedRepository_FetchOpMiss(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	var usr usermodel.User
	op := fetchOp{
		c:     cacheMock,
		codec: &usermodel.User{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return &usr, nil
		},
		logger: log.NewNopLogger(),
	}

	// when
	v, _ := op.do(context.Background())

	// then
	require.True(t, reflect.DeepEqual(v, &usr))
}
