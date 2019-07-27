/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package serializer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

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

// Serialize converts an slice of Serializer elements into its bytes representation.
func SerializeSlice(slice interface{}) ([]byte, error) {
	t := reflect.TypeOf(slice).Elem()
	if t.Kind() != reflect.Slice {
		return nil, fmt.Errorf("wrong slice type: %T", slice)
	}
	buf := bufPool.Get()
	defer bufPool.Put(buf)

	sv := reflect.ValueOf(slice).Elem()
	ln := sv.Len()

	// store slice's length
	if err := binary.Write(buf, binary.LittleEndian, int32(ln)); err != nil {
		return nil, err
	}
	// serialize elements
	for i := 0; i < ln; i++ {
		i := sv.Index(i).Addr().Interface()
		s, ok := i.(Serializer)
		if !ok {
			return nil, fmt.Errorf("element of type %T is not serializable", i)
		}
		if err := s.ToBytes(buf); err != nil {
			return nil, err
		}
	}
	res := make([]byte, buf.Len())
	copy(res, buf.Bytes())
	return res, nil
}

// Deserialize reads an entity slice of Deserilizer elements from its bytes representation.
func DeserializeSlice(b []byte, slice interface{}) error {
	t := reflect.TypeOf(slice).Elem()
	if t.Kind() != reflect.Slice {
		return fmt.Errorf("wrong slice type: %T", slice)
	}
	buf := bytes.NewBuffer(b)

	sv := reflect.ValueOf(slice).Elem()

	// read slice's length
	var ln int32
	if err := binary.Read(buf, binary.LittleEndian, &ln); err != nil {
		return err
	}
	// deserialize elements
	for i := 0; i < int(ln); i++ {
		e := reflect.New(t.Elem()).Elem()
		i := e.Addr().Interface()
		d, ok := i.(Deserializer)
		if !ok {
			return fmt.Errorf("element of type %T is not deserializable", i)
		}
		if err := d.FromBytes(buf); err != nil {
			return err
		}
		sv.Set(reflect.Append(sv, e))
	}
	return nil
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
	return deserializer.FromBytes(bytes.NewBuffer(b))
}
