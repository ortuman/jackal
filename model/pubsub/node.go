/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"bytes"
	"encoding/gob"
)

// Node represents a pubsub node
type Node struct {
	Host    string
	Name    string
	Options Options
}

// FromBytes deserializes a Node entity from its binary representation.
func (n *Node) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&n.Host); err != nil {
		return err
	}
	if err := dec.Decode(&n.Name); err != nil {
		return err
	}
	return dec.Decode(&n.Options)
}

// ToBytes converts a Node entity to its binary representation.
func (n *Node) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(n.Host); err != nil {
		return err
	}
	if err := enc.Encode(n.Name); err != nil {
		return err
	}
	return enc.Encode(n.Options)
}
