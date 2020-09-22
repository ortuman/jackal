/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const cfgExample = `
public: true
persistent: true
password_protected: false
moderated: false
allow_invites: false
allow_subject_change: true
enable_logging: true
history_length: 20
occupant_count: -1
non_anonymous: true
send_pm: "moderators"
can_get_member_list: ""
`

func TestModelRoomConfig(t *testing.T) {
	rc1 := RoomConfig{
		Public:           true,
		Persistent:       true,
		PwdProtected:     true,
		Password:         "pwd",
		Open:             true,
		Moderated:        true,
		AllowInvites:     true,
		MaxOccCnt:        20,
		AllowSubjChange:  false,
		NonAnonymous:     true,
		canSendPM:        "",
		canGetMemberList: "moderators",
	}

	buf := new(bytes.Buffer)
	require.Nil(t, rc1.ToBytes(buf))

	rc2 := RoomConfig{}
	require.Nil(t, rc2.FromBytes(buf))

	assert.EqualValues(t, rc1, rc2)
}

func TestUnmarshalYamlRoomConfig(t *testing.T) {
	badCfg := `public: "public"`
	cfg := &RoomConfig{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)

	goodCfg := cfgExample
	cfg = &RoomConfig{}
	err = yaml.Unmarshal([]byte(goodCfg), &cfg)
	require.Nil(t, err)
	require.True(t, cfg.Public)
	require.False(t, cfg.PwdProtected)
	require.False(t, cfg.Open)
	require.True(t, cfg.NonAnonymous)
	require.Equal(t, cfg.GetSendPM(), Moderators)
}

func TestSettingPrivateFieldsRoomConfig(t *testing.T) {
	cfg := &RoomConfig{}
	err := cfg.SetWhoCanSendPM("fail")
	require.NotNil(t, err)
	err = cfg.SetWhoCanSendPM(Moderators)
	require.Nil(t, err)
	require.Equal(t, Moderators, cfg.GetSendPM())

	err = cfg.SetWhoCanGetMemberList("fail")
	require.NotNil(t, err)
	err = cfg.SetWhoCanGetMemberList(None)
	require.Nil(t, err)
	require.Equal(t, None, cfg.GetCanGetMemberList())
}

func TestOccupantPermissionsRoomConfig(t *testing.T) {
	cfg := &RoomConfig{
		canSendPM:        "",
		canGetMemberList: "moderators",
	}
	o := &Occupant{
		role: moderator,
	}
	require.False(t, cfg.OccupantCanSendPM(o))
	require.True(t, cfg.OccupantCanGetMemberList(o))
}
