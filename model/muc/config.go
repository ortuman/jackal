/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"encoding/gob"
)

const (
	All = "all"

	Participants = "participants"

	Mods = "moderators"

	Visitors = "visitors"

	None = "none"
)

type RoomConfig struct {
	Public          bool
	Persistent      bool
	PwdProtected    bool
	Password        string
	Open            bool
	Moderated       bool
	RealJIDDisc     string
	SendPM          string
	AllowInvites    bool
	AllowSubjChange bool
	EnableLogging   bool
	CanGetMemberList   []string
	MaxOccCnt       int
	HistCnt         int
}

// FromBytes deserializes a RoomConfig entity from it's gob binary representation.
func (r *RoomConfig) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&r.Public); err != nil {
		return err
	}
	if err := dec.Decode(&r.Persistent); err != nil {
		return err
	}
	if err := dec.Decode(&r.PwdProtected); err != nil {
		return err
	}
	if r.PwdProtected {
		if err := dec.Decode(&r.Password); err != nil {
			return err
		}
	}
	if err := dec.Decode(&r.Open); err != nil {
		return err
	}
	if err := dec.Decode(&r.Moderated); err != nil {
		return err
	}
	if err := dec.Decode(&r.RealJIDDisc); err != nil {
		return err
	}
	if err := dec.Decode(&r.SendPM); err != nil {
		return err
	}
	if err := dec.Decode(&r.AllowInvites); err != nil {
		return err
	}
	if err := dec.Decode(&r.AllowSubjChange); err != nil {
		return err
	}
	if err := dec.Decode(&r.EnableLogging); err != nil {
		return err
	}
	if err := dec.Decode(&r.CanGetMemberList); err != nil {
		return err
	}
	if err := dec.Decode(&r.MaxOccCnt); err != nil {
		return err
	}
	if err := dec.Decode(&r.HistCnt); err != nil {
		return err
	}
	return nil
}

// ToBytes converts a RoomConfig entity to it's gob binary representation.
func (r *RoomConfig) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&r.Public); err != nil {
		return err
	}
	if err := enc.Encode(&r.Persistent); err != nil {
		return err
	}
	if err := enc.Encode(&r.PwdProtected); err != nil {
		return err
	}
	if r.PwdProtected {
		if err := enc.Encode(&r.Password); err != nil {
			return err
		}
	}
	if err := enc.Encode(&r.Open); err != nil {
		return err
	}
	if err := enc.Encode(&r.Moderated); err != nil {
		return err
	}
	if err := enc.Encode(&r.RealJIDDisc); err != nil {
		return err
	}
	if err := enc.Encode(&r.SendPM); err != nil {
		return err
	}
	if err := enc.Encode(&r.AllowInvites); err != nil {
		return err
	}
	if err := enc.Encode(&r.AllowSubjChange); err != nil {
		return err
	}
	if err := enc.Encode(&r.EnableLogging); err != nil {
		return err
	}
	if err := enc.Encode(&r.CanGetMemberList); err != nil {
		return err
	}
	if err := enc.Encode(&r.MaxOccCnt); err != nil {
		return err
	}
	if err := enc.Encode(&r.HistCnt); err != nil {
		return err
	}
	return nil
}

// NewConfigFromBytes creates and returns a new RoomConfig element from its bytes representation.
func NewConfigFromBytes(buf *bytes.Buffer) (*RoomConfig, error) {
	c := &RoomConfig{}
	if err := c.FromBytes(buf); err != nil {
		return nil, err
	}
	return c, nil
}
