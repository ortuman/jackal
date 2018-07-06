/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockListItem(t *testing.T) {
	var bi1, bi2 BlockListItem
	bi1 = BlockListItem{"ortuman", "romeo@example.net"}
	buf := new(bytes.Buffer)
	bi1.ToGob(gob.NewEncoder(buf))
	bi2.FromGob(gob.NewDecoder(buf))
	require.Equal(t, bi1, bi2)
}
