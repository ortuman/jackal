/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// canSendPM and canGetMemberList values
const (
	All        = "all"
	Moderators = "moderators"
	None       = ""
)

// RoomConfig represents different room types
type RoomConfig struct {
	Public           bool
	Persistent       bool
	PwdProtected     bool
	Password         string
	Open             bool
	Moderated        bool
	AllowInvites     bool
	MaxOccCnt        int
	AllowSubjChange  bool
	NonAnonymous     bool
	canSendPM        string
	canGetMemberList string
}

type roomConfigProxy struct {
	Public           bool   `yaml:public`
	Persistent       bool   `yaml:persistent`
	PwdProtected     bool   `yaml:password_protected`
	Open             bool   `yaml:"open"`
	Moderated        bool   `yaml:"moderated"`
	AllowInvites     bool   `yaml:"allow_invites"`
	MaxOccCnt        int    `yaml:"occupant_count"`
	NonAnonymous     bool   `yaml:"non_anonymous"`
	CanSendPM        string `yaml:"send_pm"`
	CanGetMemberList string `yaml:"can_get_member_list"`
	AllowSubjChange  bool   `yaml:"allow_subject_change"`
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
	if err := dec.Decode(&r.NonAnonymous); err != nil {
		return err
	}
	if err := dec.Decode(&r.canSendPM); err != nil {
		return err
	}
	if err := dec.Decode(&r.AllowInvites); err != nil {
		return err
	}
	if err := dec.Decode(&r.AllowSubjChange); err != nil {
		return err
	}
	if err := dec.Decode(&r.canGetMemberList); err != nil {
		return err
	}
	if err := dec.Decode(&r.MaxOccCnt); err != nil {
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
	if err := enc.Encode(&r.NonAnonymous); err != nil {
		return err
	}
	if err := enc.Encode(&r.canSendPM); err != nil {
		return err
	}
	if err := enc.Encode(&r.AllowInvites); err != nil {
		return err
	}
	if err := enc.Encode(&r.AllowSubjChange); err != nil {
		return err
	}
	if err := enc.Encode(&r.canGetMemberList); err != nil {
		return err
	}
	if err := enc.Encode(&r.MaxOccCnt); err != nil {
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

// UnmarshalYAML satisfies Unmarshaler interface, sets the default room type for the MUC service
func (r *RoomConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := roomConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	r.Public = p.Public
	r.Persistent = p.Persistent
	r.PwdProtected = p.PwdProtected
	r.Open = p.Open
	r.Moderated = p.Moderated
	r.AllowInvites = p.AllowInvites
	r.MaxOccCnt = p.MaxOccCnt
	r.AllowSubjChange = p.AllowSubjChange
	r.NonAnonymous = p.NonAnonymous
	err := r.SetWhoCanSendPM(p.CanSendPM)
	if err != nil {
		return err
	}
	err = r.SetWhoCanGetMemberList(p.CanGetMemberList)
	if err != nil {
		return err
	}
	return nil
}

func (r *RoomConfig) SetWhoCanSendPM(s string) error {
	switch s {
	case All, Moderators, None:
		r.canSendPM = s
	default:
		return fmt.Errorf("muc_config: cannot set who can send private messages to %s", s)
	}
	return nil
}

func (r *RoomConfig) WhoCanSendPM() string {
	return r.canSendPM
}

func (r *RoomConfig) OccupantCanSendPM(o *Occupant) bool {
	var hasPermission bool
	switch r.canSendPM {
	case All:
		hasPermission = true
	case None:
		hasPermission = false
	case Moderators:
		hasPermission = o.IsModerator()
	}
	return hasPermission
}

func (r *RoomConfig) SetWhoCanGetMemberList(s string) error {
	switch s {
	case All, Moderators, None:
		r.canGetMemberList = s
	default:
		return fmt.Errorf("muc_config: cannot set who can get member list to %s", s)
	}
	return nil
}

func (r *RoomConfig) WhoCanGetMemberList() string {
	return r.canGetMemberList
}

func (r *RoomConfig) OccupantCanGetMemberList(o *Occupant) bool {
	var hasPermission bool
	switch r.canGetMemberList {
	case All:
		hasPermission = true
	case None:
		hasPermission = false
	case Moderators:
		hasPermission = o.IsModerator()
	}
	return hasPermission
}

func (r *RoomConfig) OccupantCanDiscoverRealJID(o *Occupant) bool {
	if r.NonAnonymous {
		return true
	}
	return o.IsModerator()
}

func (r *RoomConfig) OccupantCanChangeSubject(o *Occupant) bool {
	if r.AllowSubjChange {
		return true
	}
	return o.IsModerator()
}
