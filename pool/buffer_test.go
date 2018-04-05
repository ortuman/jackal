/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pool

import (
	"reflect"
	"testing"

	"github.com/ortuman/jackal/util"
	"github.com/stretchr/testify/require"
)

const randomBytesLength = 256

func TestBufferPool_GetAndPut(t *testing.T) {
	p := NewBufferPool()

	buf := p.Get()
	require.Equal(t, "*bytes.Buffer", reflect.ValueOf(buf).Type().String())

	buf = p.Get()
	buf.Write(util.RandomBytes(randomBytesLength))
	require.Equal(t, randomBytesLength, buf.Len())
	p.Put(buf)
	buf = p.Get()
	require.Equal(t, 0, buf.Len())
}
