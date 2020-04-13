/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pool

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const randomBytesLength = 256

func TestBufferPool_GetAndPut(t *testing.T) {
	p := NewBufferPool()

	buf := p.Get()
	require.Equal(t, "*bytes.Buffer", reflect.ValueOf(buf).Type().String())

	buf = p.Get()

	randomBytes := make([]byte, randomBytesLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		t.Errorf("error reading random bytes: %v", err)
	}
	buf.Write(randomBytes)
	require.Equal(t, randomBytesLength, buf.Len())
	p.Put(buf)
	buf = p.Get()
	require.Equal(t, 0, buf.Len())
}
