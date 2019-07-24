/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
	"encoding/gob"
)

// Version represents a roster version info.
type Version struct {
	Ver         int
	DeletionVer int
}

// FromBytes deserializes a Version entity from its binary representation.
func (rv *Version) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&rv.Ver); err != nil {
		return err
	}
	return dec.Decode(&rv.DeletionVer)
}

// ToBytes converts a Version entity to its binary representation.
func (rv *Version) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&rv.Ver); err != nil {
		return err
	}
	return enc.Encode(&rv.DeletionVer)
}
