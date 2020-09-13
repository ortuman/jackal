/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

const (
	All = "all"

	Participants = "participants"

	Moderators = "moderators"

	Visitors = "visitors"

	None = ""
)

type RoomConfig struct {
	Public           bool
	Persistent       bool
	PwdProtected     bool
	Password         string
	Open             bool
	Moderated        bool
	AllowInvites     bool
	MaxOccCnt        int
	HistCnt          int
	AllowSubjChange  bool
	EnableLogging    bool
	realJIDDisc      string
	sendPM           string
	canGetMemberList string
}

type roomConfigProxy struct {
	Public           bool   `yaml:public`
	Persistent       bool   `yaml:persistent`
	PwdProtected     bool   `yaml:password_protected`
	Open             bool   `yaml:"open"`
	Moderated        bool   `yaml:"moderated"`
	AllowInvites     bool   `yaml:"allow_invites"`
	HistCnt          int    `yaml:"history_length"`
	MaxOccCnt        int    `yaml:"occupant_count"`
	RealJIDDisc      string `yaml:"real_jid_discovery"`
	SendPM           string `yaml:"send_pm"`
	CanGetMemberList string `yaml:"can_get_member_list"`
	AllowSubjChange  bool   `yaml:"allow_subject_change"`
	EnableLogging    bool   `yaml:"enable_logging"`
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
	if err := dec.Decode(&r.realJIDDisc); err != nil {
		return err
	}
	if err := dec.Decode(&r.sendPM); err != nil {
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
	if err := dec.Decode(&r.canGetMemberList); err != nil {
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
	if err := enc.Encode(&r.realJIDDisc); err != nil {
		return err
	}
	if err := enc.Encode(&r.sendPM); err != nil {
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
	if err := enc.Encode(&r.canGetMemberList); err != nil {
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

// Getting RoomConfig defaults for the whole service
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
	r.HistCnt = p.HistCnt
	r.MaxOccCnt = p.MaxOccCnt
	r.AllowSubjChange = p.AllowSubjChange
	r.EnableLogging = p.EnableLogging
	err := r.SetWhoCanRealJIDDisc(p.RealJIDDisc)
	if err != nil {
		return err
	}
	err = r.SetWhoCanSendPM(p.SendPM)
	if err != nil {
		return err
	}
	err = r.SetWhoCanGetMemberList(p.CanGetMemberList)
	if err != nil {
		return err
	}
	return nil
}

func (r *RoomConfig) SetWhoCanRealJIDDisc(s string) error {
	switch s {
	case All, Moderators, None:
		r.realJIDDisc = s
	default:
		return fmt.Errorf("muc_config: cannot set who can discover real JIDs to %s", s)
	}
	return nil
}

func (r *RoomConfig) GetRealJIDDisc() string {
	return r.realJIDDisc
}

func (r *RoomConfig) OccupantCanDiscoverRealJID(o *Occupant) bool {
	var hasPermission bool
	switch r.realJIDDisc {
	case All:
		hasPermission = true
	case None:
		hasPermission = false
	case Moderators:
		hasPermission = o.IsModerator()
	}
	return hasPermission
}

func (r *RoomConfig) SetWhoCanSendPM(s string) error {
	switch s {
	case All, Moderators, None:
		r.sendPM = s
	default:
		return fmt.Errorf("muc_config: cannot set who can send private messages to %s", s)
	}
	return nil
}

func (r *RoomConfig) GetSendPM() string {
	return r.sendPM
}

func (r *RoomConfig) OccupantCanSendPM(o *Occupant) bool {
	var hasPermission bool
	switch r.sendPM {
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

func (r *RoomConfig) GetCanGetMemberList() string {
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
