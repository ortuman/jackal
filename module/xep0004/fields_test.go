package xep0004

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFields_ValueForField(t *testing.T) {
	f := Fields{
		{
			Var:    "var1",
			Values: []string{"foo"},
		},
	}
	require.Equal(t, "foo", f.ValueForField("var1"))
	require.Equal(t, "", f.ValueForField("var2"))
}
