/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import "encoding/gob"

// Version represents a roster version info.
type Version struct {
	Ver         int
	DeletionVer int
}

// FromGob deserializes a Version entity
// from it's gob binary representation.
func (rv *Version) FromGob(dec *gob.Decoder) error {
	dec.Decode(&rv.Ver)
	dec.Decode(&rv.DeletionVer)
	return nil
}

// ToGob converts a Version entity
// to it's gob binary representation.
func (rv *Version) ToGob(enc *gob.Encoder) {
	enc.Encode(&rv.Ver)
	enc.Encode(&rv.DeletionVer)
}
