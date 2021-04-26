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

	"github.com/stretchr/testify/require"
)

func TestDialback_Key(t *testing.T) {
	// given
	secret := "s3cr3tf0rd14lb4ck"
	sender := "example.org"
	target := "xmpp.example.com"
	streamID := "D60000229F"

	// when
	dbK := dbKey(secret, sender, target, streamID)

	// then
	require.Equal(t, "37c69b1cf07a3f67c04a5ef5902fa5114f2c76fe4a2686482ba5b89323075643", dbK)
}

func TestDialback_RegisterRequest(t *testing.T) {
	// given
	kvMock := &kvMock{}

	var pKey, pVal string
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		pKey = key
		pVal = value
		return nil
	}
	// when
	sender := "example.org"
	target := "xmpp.example.com"
	streamID := "D60000229F"

	// then
	_ = registerDbRequest(context.Background(), sender, target, streamID, kvMock)

	require.Equal(t, "db://D60000229F", pKey)
	require.Equal(t, "example.org xmpp.example.com", pVal)
}

func TestDialback_IsRequestOn(t *testing.T) {
	// given
	kvMock := &kvMock{}

	var pKey string
	kvMock.GetFunc = func(ctx context.Context, key string) ([]byte, error) {
		pKey = key
		return []byte("example.org xmpp.example.com"), nil
	}
	// when
	sender := "example.org"
	target := "xmpp.example.com"
	streamID := "D60000229F"

	on, _ := isDbRequestOn(context.Background(), sender, target, streamID, kvMock)

	// then
	require.Equal(t, "db://D60000229F", pKey)
	require.True(t, on)
}

func TestDialback_UnregisterRequest(t *testing.T) {
	// given
	kvMock := &kvMock{}

	var pKey string
	kvMock.DelFunc = func(ctx context.Context, key string) error {
		pKey = key
		return nil
	}
	// when
	streamID := "D60000229F"

	// then
	_ = unregisterDbRequest(context.Background(), streamID, kvMock)

	require.Equal(t, "db://D60000229F", pKey)
}
