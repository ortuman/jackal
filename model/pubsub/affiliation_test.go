package pubsubmodel

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliation_Serialize(t *testing.T) {
	a := Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "owner",
	}
	b := bytes.NewBuffer(nil)
	require.Nil(t, a.ToBytes(b))

	var a2 Affiliation
	require.Nil(t, a2.FromBytes(b))

	require.True(t, reflect.DeepEqual(a, a2))
}
