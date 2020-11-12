/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestMySQLStorageInsertRoom(t *testing.T) {
	room := getTestRoom()
	s, mock := newRoomMock()
	rc := room.Config
	userJID := room.GetAllUserJIDs()[0]
	occJID, _ := room.GetOccupantJID(&userJID)
	invitedUser := room.GetAllInvitedUsers()[0]

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rooms (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(room.RoomJID.String(), room.Name, room.Desc, room.Subject, room.Language,
			room.Locked, room.GetOccupantsOnlineCount(), room.Name, room.Desc, room.Subject,
			room.Language, room.Locked, room.GetOccupantsOnlineCount()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO rooms_config (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(room.RoomJID.String(), rc.Public, rc.Persistent, rc.PwdProtected,
			rc.Password, rc.Open, rc.Moderated, rc.AllowInvites, rc.MaxOccCnt, rc.AllowSubjChange,
			rc.NonAnonymous, rc.GetSendPM(), rc.WhoCanGetMemberList(), rc.Public, rc.Persistent,
			rc.PwdProtected, rc.Password, rc.Open, rc.Moderated, rc.AllowInvites, rc.MaxOccCnt,
			rc.AllowSubjChange, rc.NonAnonymous, rc.GetSendPM(), rc.WhoCanGetMemberList()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO rooms_invites (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(room.RoomJID.String(), invitedUser, invitedUser).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO rooms_users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(room.RoomJID.String(), userJID.String(), occJID.String(), occJID.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := s.UpsertRoom(context.Background(), room)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRoomMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rooms (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(room.RoomJID.String(), room.Name, room.Desc, room.Subject, room.Language,
			room.Locked, room.GetOccupantsOnlineCount(), room.Name, room.Desc, room.Subject,
			room.Language, room.Locked, room.GetOccupantsOnlineCount()).
		WillReturnError(errMocked)
	mock.ExpectRollback()

	err = s.UpsertRoom(context.Background(), room)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, err, errMocked)
}

func TestMySQLStorageFetchRoom(t *testing.T) {
	room := getTestRoom()
	rc := room.Config
	s, mock := newRoomMock()
	roomColumns := []string{"room_jid", "name", "description", "subject", "language", "locked",
		"occupants_online"}
	rcColumns := []string{"room_jid", "public", "persistent", "pwd_protected", "password", "open",
		"moderated", "allow_invites", "max_occupants", "allow_subj_change", "non_anonymous",
		"can_send_pm", "can_get_member_list"}
	usersColumns := []string{"room_jid", "user_jid", "occupant_jid"}
	invitesColumns := []string{"room_jid", "user_jid"}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(roomColumns))
	mock.ExpectCommit()

	r, _ := s.FetchRoom(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, r)

	s, mock = newRoomMock()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(roomColumns).
			AddRow(room.RoomJID.String(), room.Name, room.Desc, room.Subject, room.Language,
				room.Locked, room.GetOccupantsOnlineCount()))
	mock.ExpectQuery("SELECT (.+) FROM rooms_config (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(rcColumns).
			AddRow(room.RoomJID.String(), rc.Public, rc.Persistent, rc.PwdProtected, rc.Password,
				rc.Open, rc.Moderated, rc.AllowInvites, rc.MaxOccCnt, rc.AllowSubjChange,
				rc.NonAnonymous, rc.GetSendPM(), rc.WhoCanGetMemberList()))
	mock.ExpectQuery("SELECT (.+) FROM rooms_users (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(usersColumns).
			AddRow(room.RoomJID.String(), room.GetAllUserJIDs()[0].String(),
				room.GetAllOccupantJIDs()[0].String()))
	mock.ExpectQuery("SELECT (.+) FROM rooms_invites (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(invitesColumns).
			AddRow(room.RoomJID.String(), room.GetAllInvitedUsers()[0]))
	mock.ExpectCommit()
	r, err := s.FetchRoom(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, r)
	assert.EqualValues(t, room, r)

	s, mock = newRoomMock()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).WillReturnError(errMocked)
	mock.ExpectRollback()
	_, err = s.FetchRoom(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestMySQLStorageDeleteRoom(t *testing.T) {
	room := getTestRoom()
	s, mock := newRoomMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM rooms_config (.+)").
		WithArgs(room.RoomJID.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM rooms_users (.+)").
		WithArgs(room.RoomJID.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM rooms_invites (.+)").
		WithArgs(room.RoomJID.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteRoom(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRoomMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).WillReturnError(errMocked)
	mock.ExpectRollback()

	err = s.DeleteRoom(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestMySQLStorageRoomExists(t *testing.T) {
	room := getTestRoom()
	countCols := []string{"count"}

	s, mock := newRoomMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnRows(sqlmock.NewRows(countCols).AddRow(1))

	ok, err := s.RoomExists(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = newRoomMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM rooms (.+)").
		WithArgs(room.RoomJID.String()).
		WillReturnError(errMocked)
	_, err = s.RoomExists(context.Background(), room.RoomJID)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func newRoomMock() (*mySQLRoom, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLRoom{
		mySQLStorage: s,
	}, sqlMock
}

func getTestRoom() *mucmodel.Room {
	rc := mucmodel.RoomConfig{
		Public:       true,
		Persistent:   true,
		PwdProtected: false,
		Open:         true,
		Moderated:    false,
	}
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)

	r := &mucmodel.Room{
		Name:    "testRoom",
		RoomJID: j,
		Desc:    "Room for Testing",
		Config:  &rc,
		Locked:  false,
	}

	oJID, _ := jid.NewWithString("testroom@conference.jackal.im/owner", true)
	owner, _ := mucmodel.NewOccupant(oJID, oJID.ToBareJID())
	r.AddOccupant(owner)
	r.InviteUser(oJID.ToBareJID())

	return r
}
