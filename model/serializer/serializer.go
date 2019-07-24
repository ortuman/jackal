/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package serializer

import (
	"bytes"

	"github.com/ortuman/jackal/pool"
)

var bufPool = pool.NewBufferPool()

// Serializer represents a Gob serializable entity.
type Serializer interface {
	ToBytes(buf *bytes.Buffer) error
}

// Deserializer represents a Gob deserializable entity.
type Deserializer interface {
	FromBytes(buf *bytes.Buffer) error
}

// Serialize converts a serializable entity into its bytes representation.
func Serialize(serializer Serializer) ([]byte, error) {
	buf := bufPool.Get()
	defer bufPool.Put(buf)

	if err := serializer.ToBytes(buf); err != nil {
		return nil, err
	}
	res := make([]byte, buf.Len())
	copy(res, buf.Bytes())

	return res, nil
}

// Deserialize reads an entity from its bytes representation.
func Deserialize(b []byte, deserializer Deserializer) error {
	buf := bufPool.Get()
	defer bufPool.Put(buf)

	buf.Write(b)
	return deserializer.FromBytes(buf)
}
