/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElementSerialization(t *testing.T) {
	n := NewElementNamespace("el1", "exodus:ns")
	n.SetText("a simple text")
	n.AppendElement(NewElementName("a1"))
	n.AppendElement(NewElementName("b2"))
	n.AppendElement(NewElementName("b3"))

	buf := new(bytes.Buffer)
	n.ToBytes(buf)

	var n2 MutableElement
	n2.FromBytes(buf)

	require.Equal(t, n.String(), n2.String())
}
