/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"encoding/gob"
)

// BlockListItem represents block list item storage entity.
type BlockListItem struct {
	Username string
	JID      string
}

// FromBytes deserializes a BlockListItem entity from its binary representation.
func (bli *BlockListItem) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&bli.Username); err != nil {
		return err
	}
	return dec.Decode(&bli.JID)
}

// ToBytes converts a BlockListItem entity to its binary representation.
func (bli *BlockListItem) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&bli.Username); err != nil {
		return err
	}
	return enc.Encode(&bli.JID)
}
