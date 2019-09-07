/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode_Serialization(t *testing.T) {
	n := Node{}
	n.Name = "playing_lists"
	n.Host = "jackal.im"

	n.Options.Title = "Playing lists"
	n.Options.NotifySub = true

	buf := bytes.NewBuffer(nil)
	require.Nil(t, n.ToBytes(buf))

	n2 := Node{}
	_ = n2.FromBytes(buf)

	require.True(t, reflect.DeepEqual(&n, &n2))
}
