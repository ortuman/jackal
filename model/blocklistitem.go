/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import "encoding/gob"

// BlockListItem represents block list item storage entity.
type BlockListItem struct {
	Username string
	JID      string
}

// FromGob deserializes a BlockListItem entity from it's gob binary representation.
func (bli *BlockListItem) FromGob(dec *gob.Decoder) error {
	dec.Decode(&bli.Username)
	dec.Decode(&bli.JID)
	return nil
}

// ToGob converts a BlockListItem entity
// to it's gob binary representation.
func (bli *BlockListItem) ToGob(enc *gob.Encoder) {
	enc.Encode(&bli.Username)
	enc.Encode(&bli.JID)
}
