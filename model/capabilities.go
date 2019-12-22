/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"encoding/gob"
)

// Capabilities represents presence capabilities info
type Capabilities struct {
	Node     string
	Ver      string
	Features []string
}

// FromBytes deserializes a Capabilities entity from its binary representation.
func (c *Capabilities) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&c.Node); err != nil {
		return err
	}
	if err := dec.Decode(&c.Ver); err != nil {
		return err
	}
	return dec.Decode(&c.Features)
}

// ToBytes converts a Capabilities entity to its binary representation.
func (c *Capabilities) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&c.Node); err != nil {
		return err
	}
	if err := enc.Encode(&c.Ver); err != nil {
		return err
	}
	return enc.Encode(&c.Features)
}

// HasFeature returns whether or not Capabilities instance contains a concrete feature
func (c *Capabilities) HasFeature(feature string) bool {
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}
