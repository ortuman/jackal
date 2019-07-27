/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockListItem(t *testing.T) {
	var bi1, bi2 BlockListItem
	bi1 = BlockListItem{"ortuman", "romeo@example.net"}
	buf := new(bytes.Buffer)
	require.Nil(t, bi1.ToBytes(buf))
	require.Nil(t, bi2.FromBytes(buf))
	require.Equal(t, bi1, bi2)
}
