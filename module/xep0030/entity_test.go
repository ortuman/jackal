/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEntity_Features(t *testing.T) {
	e := &Entity{}
	e.AddFeature("b")
	e.AddFeature("c")
	e.AddFeature("a")
	require.Equal(t, []string{"a", "b", "c"}, e.Features())
	e.RemoveFeature("b")
	require.Equal(t, []string{"a", "c"}, e.Features())
}

func TestEntity_Identities(t *testing.T) {
	e := &Entity{}
	idns := []Identity{
		{"c0", "t0", "n0"},
		{"c1", "t1", "n1"},
		{"c2", "t2", "n2"},
	}
	e.AddIdentity(idns[0])
	e.AddIdentity(idns[1])
	e.AddIdentity(idns[2])
	require.Equal(t, idns, e.Identities())
	e.RemoveIdentity(idns[1])
	require.Equal(t, []Identity{idns[0], idns[2]}, e.Identities())
}

func TestEntity_Items(t *testing.T) {
	e := &Entity{}
	itms := []Item{
		{"j0", "n0", "n0"},
		{"j1", "n1", "n1"},
		{"j2", "n2", "n2"},
	}
	e.AddItem(itms[0])
	e.AddItem(itms[1])
	e.AddItem(itms[2])
	require.Equal(t, itms, e.Items())
	e.RemoveItem(itms[1])
	require.Equal(t, []Item{itms[0], itms[2]}, e.Items())
}
