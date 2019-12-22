/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"bytes"
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
)

// Item represents a pubsub node item
type Item struct {
	ID        string
	Publisher string
	Payload   xmpp.XElement
}

// FromBytes deserializes a Item entity from its binary representation.
func (i *Item) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&i.ID); err != nil {
		return err
	}
	if err := dec.Decode(&i.Publisher); err != nil {
		return err
	}
	var hasPayload bool
	if err := dec.Decode(&hasPayload); err != nil {
		return err
	}
	if hasPayload {
		var elem xmpp.Element
		if err := elem.FromBytes(buf); err != nil {
			return err
		}
		i.Payload = &elem
	}
	return nil
}

// ToBytes converts a Item entity to its binary representation.
func (i *Item) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(i.ID); err != nil {
		return err
	}
	if err := enc.Encode(i.Publisher); err != nil {
		return err
	}
	hasPayload := i.Payload != nil
	if err := enc.Encode(hasPayload); err != nil {
		return err
	}
	if i.Payload != nil {
		return i.Payload.ToBytes(buf)
	}
	return nil
}
