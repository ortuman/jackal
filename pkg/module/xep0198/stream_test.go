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

package xep0198

import (
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/stretchr/testify/require"
)

func TestEncodeSMID(t *testing.T) {
	// given
	jd, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	nonce := make([]byte, 16)
	for i := range nonce {
		nonce[i] = byte(i)
	}

	// when
	smID := encodeSMID(jd, nonce)

	// then
	require.Equal(t, "b3J0dW1hbkBqYWNrYWwuaW0veWFyZAAAAQIDBAUGBwgJCgsMDQ4P", smID)
}

func TestDecodeSMID(t *testing.T) {
	// given
	smID := "b3J0dW1hbkBqYWNrYWwuaW0veWFyZAAAAQIDBAUGBwgJCgsMDQ4P"

	// when
	jd, nonce, err := decodeSMID(smID)

	// then
	require.Nil(t, err)
	require.NotNil(t, jd)
	require.NotNil(t, nonce)
}

func TestDecodeSMIDError(t *testing.T) {
	// given
	badID := "fooID"

	// when
	jd, nonce, err := decodeSMID(badID)

	// then
	require.NotNil(t, err)
	require.Nil(t, jd)
	require.Nil(t, nonce)
}
