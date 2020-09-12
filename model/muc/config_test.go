/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

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
real_jid_discovery: "all"
send_private_messages: "moderators"
can_get_member_list: "none"
`

func TestModelRoomConfig(t *testing.T){
	rc1 := RoomConfig{
		Public: true,
		Persistent: true,
		PwdProtected: true,
		Password: "pwd",
		Open: true,
		Moderated: true,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, rc1.ToBytes(buf))

	rc2 := RoomConfig{}
	require.Nil(t, rc2.FromBytes(buf))
	require.Equal(t, rc1.Public, rc2.Public)
	require.Equal(t, rc1.Persistent, rc2.Persistent)
	require.Equal(t, rc1.PwdProtected, rc2.PwdProtected)
	require.Equal(t, rc1.Password, rc2.Password)
	require.Equal(t, rc1.Open, rc2.Open)
	require.Equal(t, rc1.Moderated, rc2.Moderated)
}

func TestUnmarshalYamlRoomConfig(t *testing.T){
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
}
