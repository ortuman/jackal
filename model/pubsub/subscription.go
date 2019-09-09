package pubsubmodel

import (
	"bytes"
	"encoding/gob"
)

type Subscription struct {
	SubID        string
	JID          string
	Subscription string
}

// FromBytes deserializes a Subscription entity from its binary representation.
func (s *Subscription) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&s.SubID); err != nil {
		return err
	}
	if err := dec.Decode(&s.JID); err != nil {
		return err
	}
	return dec.Decode(&s.Subscription)
}

// ToBytes converts a Subscription entity to its binary representation.
func (s *Subscription) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(s.SubID); err != nil {
		return err
	}
	if err := enc.Encode(s.JID); err != nil {
		return err
	}
	return enc.Encode(s.Subscription)
}
