/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package capsmodel

import (
	"bytes"
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
)

// PresenceCaps represents the combination of along with its capabilities.
type PresenceCaps struct {
	Presence *xmpp.Presence
	Caps     *Capabilities
}

// FromBytes deserializes a Capabilities entity from its binary representation.
func (p *PresenceCaps) FromBytes(buf *bytes.Buffer) error {
	presence, err := xmpp.NewPresenceFromBytes(buf)
	if err != nil {
		return err
	}
	var hasCaps bool

	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&hasCaps); err != nil {
		return err
	}
	p.Presence = presence
	if hasCaps {
		return dec.Decode(&p.Caps)
	}
	return nil
}

// ToBytes converts a Capabilities entity to its binary representation.
func (p *PresenceCaps) ToBytes(buf *bytes.Buffer) error {
	if err := p.Presence.ToBytes(buf); err != nil {
		return err
	}
	enc := gob.NewEncoder(buf)

	hasCaps := p.Caps != nil
	if err := enc.Encode(hasCaps); err != nil {
		return err
	}
	if p.Caps != nil {
		if err := enc.Encode(p.Caps); err != nil {
			return err
		}
	}
	return nil
}
