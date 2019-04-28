package xep0004

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFields_BoolForField(t *testing.T) {
	f := Fields{
		{
			Var:    "var1",
			Values: []string{"true"},
		},
		{
			Var:    "var2",
			Values: []string{"1"},
		},
		{
			Var:    "var3",
			Values: []string{"0"},
		},
		{
			Var:    "var4",
			Values: []string{"foo"},
		},
	}
	require.True(t, f.BoolForField("var1"))
	require.True(t, f.BoolForField("var2"))
	require.False(t, f.BoolForField("var3"))
	require.False(t, f.BoolForField("var4"))
}

func TestFields_IntForField(t *testing.T) {
	f := Fields{
		{
			Var:    "var1",
			Values: []string{"4096"},
		},
		{
			Var:    "var2",
			Values: []string{"foo"},
		},
	}
	require.Equal(t, 4096, f.IntForField("var1"))
	require.Equal(t, 0, f.IntForField("var2"))
}

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
