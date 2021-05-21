// Copyright 2020 The jackal Authors
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

package c2smodel

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfo_SetGet(t *testing.T) {
	// given
	m := NewMutableInfo()

	// when
	m.SetString("k1", "v1")
	m.SetBool("k2", true)
	m.SetInt("k3", 46)
	m.SetFloat("k4", 2.24532)

	allKeys := m.AllKeys()
	sort.Slice(allKeys, func(i, j int) bool { return allKeys[i] < allKeys[j] })

	k4v, _ := m.Value("k4")

	cpInf := m.Copy()

	// then
	require.Equal(t, "v1", m.String("k1"))
	require.Equal(t, true, m.Bool("k2"))
	require.Equal(t, 46, m.Int("k3"))
	require.Equal(t, 2.24532, m.Float("k4"))

	require.Equal(t, "2.24532E+00", k4v)

	require.Equal(t, []string{"k1", "k2", "k3", "k4"}, allKeys)

	require.True(t, reflect.DeepEqual(cpInf.m, m.m))
}

func TestInfo_Value(t *testing.T) {
	// given
	// when
	// then
}
