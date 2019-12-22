/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestItem_Serialization(t *testing.T) {
	it := Item{}
	it.ID = "1234"
	it.Publisher = "ortuman@jackal.im"
	it.Payload = xmpp.NewElementName("el")

	buf := bytes.NewBuffer(nil)
	require.Nil(t, it.ToBytes(buf))

	it2 := Item{}
	_ = it2.FromBytes(buf)

	require.True(t, reflect.DeepEqual(&it, &it2))

	// nil payload
	it.Payload = nil

	buf2 := bytes.NewBuffer(nil)
	require.Nil(t, it.ToBytes(buf2))

	it3 := Item{}
	_ = it3.FromBytes(buf2)

	require.True(t, reflect.DeepEqual(&it, &it3))
}
