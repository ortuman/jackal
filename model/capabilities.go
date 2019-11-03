/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"encoding/gob"
)

type Capabilities struct {
	Features []string
}

// FromBytes deserializes a Capabilities entity from its binary representation.
func (c *Capabilities) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	return dec.Decode(&c.Features)
}

// ToBytes converts a Capabilities entity to its binary representation.
func (c *Capabilities) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	return enc.Encode(&c.Features)
}
