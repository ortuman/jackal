// Copyright 2021 The jackal Authors
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

package rediscache

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/require"
)

func TestRedisCache_GetHit(t *testing.T) {
	// given
	db, mock := redismock.NewClientMock()

	// when
	mock.ExpectGet("k0").SetVal("1234")

	c := New(db)
	b, err := c.Get(context.Background(), "k0")

	// then
	require.Nil(t, err)
	require.Equal(t, "1234", string(b))

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestRedisCache_GetMiss(t *testing.T) {
	// given
	db, mock := redismock.NewClientMock()

	// when
	mock.ExpectGet("k0").RedisNil()

	c := New(db)
	b, err := c.Get(context.Background(), "k0")

	// then
	require.Nil(t, err)
	require.Nil(t, b)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestRedisCache_Set(t *testing.T) {
	// given
	db, mock := redismock.NewClientMock()

	// when
	mock.ExpectSet("k0", []byte("1234"), 0).SetVal("OK")

	c := New(db)
	err := c.Set(context.Background(), "k0", []byte("1234"))

	// then
	require.Nil(t, err)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestRedisCache_Del(t *testing.T) {
	// given
	db, mock := redismock.NewClientMock()

	// when
	mock.ExpectDel("k0").SetVal(1)

	c := New(db)
	err := c.Del(context.Background(), "k0")

	// then
	require.Nil(t, err)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestRedisCache_Exists(t *testing.T) {
	// given
	db, mock := redismock.NewClientMock()

	// when
	mock.ExpectDel("k0").SetVal(1)

	c := New(db)
	err := c.Del(context.Background(), "k0")

	// then
	require.Nil(t, err)

	require.Nil(t, mock.ExpectationsWereMet())
}
