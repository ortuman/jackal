/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

type mySQLRoom struct {
	*mySQLStorage
}

func newRoom(db *sql.DB) *mySQLRoom {
	return &mySQLRoom{
		mySQLStorage: newStorage(db),
	}
}

func (r *mySQLRoom) UpsertRoom(ctx context.Context, room *mucmodel.Room) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		// rooms table
		columns := []string{"room_jid", "name", "description", "subject", "language", "locked",
			"occupants_online"}
		values := []interface{}{room.RoomJID.String(), room.Name, room.Desc, room.Subject,
			room.Language, room.Locked, room.GetOccupantsOnlineCount()}
		q := sq.Insert("rooms").
			Columns(columns...).
			Values(values...).
			Suffix("ON DUPLICATE KEY UPDATE name = ?, description = ?, subject = ?, language = ?,"+
				" locked = ?, occupants_online = ?", room.Name, room.Desc, room.Subject,
				room.Language, room.Locked, room.GetOccupantsOnlineCount())
		_, err := q.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// rooms_config table
		rc := room.Config
		columns = []string{"room_jid", "public", "persistent", "pwd_protected", "password", "open",
			"moderated", "allow_invites", "max_occupants", "allow_subj_change", "non_anonymous",
			"can_send_pm", "can_get_member_list"}
		values = []interface{}{room.RoomJID.String(), rc.Public, rc.Persistent, rc.PwdProtected,
			rc.Password, rc.Open, rc.Moderated, rc.AllowInvites, rc.MaxOccCnt, rc.AllowSubjChange,
			rc.NonAnonymous, rc.GetSendPM(), rc.WhoCanGetMemberList()}
		q = sq.Insert("rooms_config").
			Columns(columns...).
			Values(values...).
			Suffix("ON DUPLICATE KEY UPDATE public = ?, persistent = ?, pwd_protected = ?, "+
				"password = ?, open = ?, moderated = ?, allow_invites = ?, max_occupants = ?, "+
				"allow_subj_change = ?, non_anonymous = ?, can_send_pm = ?, can_get_member_list",
				rc.Public, rc.Persistent, rc.PwdProtected, rc.Password, rc.Open, rc.Moderated,
				rc.AllowInvites, rc.MaxOccCnt, rc.AllowSubjChange, rc.NonAnonymous, rc.GetSendPM(),
				rc.WhoCanGetMemberList())
		_, err = q.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}

		// rooms_invites table
		columns = []string{"room_jid", "user_jid"}
		for _, u := range room.GetAllInvitedUsers() {
			values = []interface{}{room.RoomJID.String(), u}
			q = sq.Insert("rooms_invites").
				Columns(columns...).
				Values(values...).
				Suffix("ON DUPLICATE KEY UPDATE user_jid = ?", u)
			_, err = q.RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}

		// rooms_users table
		columns = []string{"room_jid", "user_jid", "occupant_jid"}
		for _, u := range room.GetAllUserJIDs() {
			occJID, _ := room.GetOccupantJID(&u)
			values = []interface{}{room.RoomJID.String(), u.String(), occJID.String()}
			q = sq.Insert("rooms_users").
				Columns(columns...).
				Values(values...).
				Suffix("ON DUPLICATE KEY UPDATE occupant_jid = ?", occJID.String())
			_, err = q.RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *mySQLRoom) FetchRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	room := &mucmodel.Room{}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// fetch room data
	q := sq.Select("room_jid", "name", "description", "subject", "language", "locked",
		"occupants_online").
		From("rooms").
		Where(sq.Eq{"room_jid": roomJID.String()})
	var onlineCnt int
	var roomJIDStr string
	err = q.RunWith(tx).
		QueryRowContext(ctx).
		Scan(&roomJIDStr, &room.Name, &room.Desc, &room.Subject, &room.Language, &room.Locked,
			&onlineCnt)
	switch err {
	case nil:
		rJID, err := jid.NewWithString(roomJIDStr, false)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		room.RoomJID = rJID
		room.SetOccupantsOnlineCount(onlineCnt)
	case sql.ErrNoRows:
		_ = tx.Commit()
		return nil, nil
	default:
		_ = tx.Rollback()
		return nil, err
	}

	// fetchRoomConfig
	rc := &mucmodel.RoomConfig{}
	q = sq.Select("room_jid", "public", "persistent", "pwd_protected", "password", "open",
		"moderated", "allow_invites", "max_occupants", "allow_subj_change", "non_anonymous",
		"can_send_pm", "can_get_member_list").
		From("rooms_config").
		Where(sq.Eq{"room_jid": roomJID.String()})
	var dummy, sendPM, membList string
	err = q.RunWith(tx).
		QueryRowContext(ctx).
		Scan(&dummy, &rc.Public, &rc.Persistent, &rc.PwdProtected, &rc.Password, &rc.Open,
			&rc.Moderated, &rc.AllowInvites, &rc.MaxOccCnt, &rc.AllowSubjChange, &rc.NonAnonymous,
			&sendPM, &membList)
	switch err {
	case nil:
		err = rc.SetWhoCanSendPM(sendPM)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		err = rc.SetWhoCanGetMemberList(membList)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	case sql.ErrNoRows:
		_ = tx.Commit()
		return nil, nil
	default:
		_ = tx.Rollback()
		return nil, err
	}
	room.Config = rc

	// fetch users in the room
	res, err := sq.Select("room_jid", "user_jid", "occupant_jid").
		From("rooms_users").
		Where(sq.Eq{"room_jid": roomJID.String()}).
		RunWith(tx).QueryContext(ctx)
	for res.Next() {
		var uJIDStr, oJIDStr string
		if err := res.Scan(&dummy, &uJIDStr, &oJIDStr); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		uJID, err := jid.NewWithString(uJIDStr, false)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		oJID, err := jid.NewWithString(oJIDStr, false)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		err = room.MapUserToOccupantJID(uJID, oJID)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	// fetch invited users
	resInv, err := sq.Select("room_jid", "user_jid").
		From("rooms_invites").
		Where(sq.Eq{"room_jid": roomJID.String()}).
		RunWith(tx).QueryContext(ctx)
	for resInv.Next() {
		var uJIDStr string
		if err := resInv.Scan(&dummy, &uJIDStr); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		uJID, err := jid.NewWithString(uJIDStr, false)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		err = room.InviteUser(uJID)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *mySQLRoom) DeleteRoom(ctx context.Context, roomJID *jid.JID) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		_, err := sq.Delete("rooms").Where(sq.Eq{"room_jid": roomJID.String()}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("rooms_config").Where(sq.Eq{"room_jid": roomJID.String()}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("rooms_users").Where(sq.Eq{"room_jid": roomJID.String()}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("rooms_invites").Where(sq.Eq{"room_jid": roomJID.String()}).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *mySQLRoom) RoomExists(ctx context.Context, roomJID *jid.JID) (bool, error) {
	q := sq.Select("COUNT(*)").
		From("rooms").
		Where(sq.Eq{"room_jid": roomJID.String()})

	var count int
	err := q.RunWith(r.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
