/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

type pgSQLRoom struct {
	*pgSQLStorage
}

func newRoom(db *sql.DB) *pgSQLRoom {
	return &pgSQLRoom{
		pgSQLStorage: newStorage(db),
	}
}

func (r *pgSQLRoom) UpsertRoom(ctx context.Context, room *mucmodel.Room) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		// rooms table
		columns := []string{"room_jid", "name", "description", "subject", "language", "locked",
			"occupants_online"}
		values := []interface{}{room.RoomJID.String(), room.Name, room.Desc, room.Subject,
			room.Language, room.Locked, room.GetOccupantsOnlineCount()}
		q := sq.Insert("rooms").
			Columns(columns...).
			Values(values...).
			Suffix("ON CONFLICT (room_jid) DO UPDATE SET name = $2, description = $3, subject = $4" + ", language = $5, locked = $6, occupants_online = $7")
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
			rc.NonAnonymous, rc.WhoCanSendPM(), rc.WhoCanGetMemberList()}
		q = sq.Insert("rooms_config").
			Columns(columns...).
			Values(values...).
			Suffix("ON CONFLICT (room_jid) DO UPDATE SET public = $2, persistent = $3, pwd_protected = $4, " +
				"password = $5, open = $6, moderated = $7, allow_invites = $8, max_occupants = $9, " +
				"allow_subj_change = $10, non_anonymous = $11, can_send_pm = $12, can_get_member_list = $13")
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
				Suffix("ON CONFLICT (room_jid) DO UPDATE SET user_jid = $2")
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
				Suffix("ON CONFLICT (room_jid) DO UPDATE SET occupant_jid = $3")
			_, err = q.RunWith(tx).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *pgSQLRoom) FetchRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	room, err := fetchRoomData(ctx, tx, roomJID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		_ = tx.Commit()
		return nil, nil
	default:
		_ = tx.Rollback()
		return nil, err
	}

	err = fetchRoomConfig(ctx, tx, room, roomJID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		_ = tx.Commit()
		return nil, nil
	default:
		_ = tx.Rollback()
		return nil, err
	}

	err = fetchRoomUsers(ctx, tx, room, roomJID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	err = fetchRoomInvites(ctx, tx, room, roomJID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return room, nil
}

func fetchRoomData(ctx context.Context, tx *sql.Tx, roomJID *jid.JID) (*mucmodel.Room,
	error) {
	room := &mucmodel.Room{}
	// fetch room data
	q := sq.Select("room_jid", "name", "description", "subject", "language", "locked",
		"occupants_online").
		From("rooms").
		Where(sq.Eq{"room_jid": roomJID.String()})
	var onlineCnt int
	var roomJIDStr string
	err := q.RunWith(tx).
		QueryRowContext(ctx).
		Scan(&roomJIDStr, &room.Name, &room.Desc, &room.Subject, &room.Language, &room.Locked,
			&onlineCnt)
	switch err {
	case nil:
		rJID, err := jid.NewWithString(roomJIDStr, false)
		if err != nil {
			return nil, err
		}
		room.RoomJID = rJID
		room.SetOccupantsOnlineCount(onlineCnt)
	default:
		return nil, err
	}
	return room, nil
}

func fetchRoomConfig(ctx context.Context, tx *sql.Tx, room *mucmodel.Room,
	roomJID *jid.JID) error {
	rc := &mucmodel.RoomConfig{}
	q := sq.Select("room_jid", "public", "persistent", "pwd_protected", "password", "open",
		"moderated", "allow_invites", "max_occupants", "allow_subj_change", "non_anonymous",
		"can_send_pm", "can_get_member_list").
		From("rooms_config").
		Where(sq.Eq{"room_jid": roomJID.String()})
	var dummy, sendPM, membList string
	err := q.RunWith(tx).
		QueryRowContext(ctx).
		Scan(&dummy, &rc.Public, &rc.Persistent, &rc.PwdProtected, &rc.Password, &rc.Open,
			&rc.Moderated, &rc.AllowInvites, &rc.MaxOccCnt, &rc.AllowSubjChange, &rc.NonAnonymous,
			&sendPM, &membList)
	switch err {
	case nil:
		err = rc.SetWhoCanSendPM(sendPM)
		if err != nil {
			return err
		}
		err = rc.SetWhoCanGetMemberList(membList)
		if err != nil {
			return err
		}
	default:
		return err
	}
	room.Config = rc
	return nil
}

func fetchRoomUsers(ctx context.Context, tx *sql.Tx, room *mucmodel.Room,
	roomJID *jid.JID) error {
	res, err := sq.Select("room_jid", "user_jid", "occupant_jid").
		From("rooms_users").
		Where(sq.Eq{"room_jid": roomJID.String()}).
		RunWith(tx).QueryContext(ctx)
	if err != nil {
		return err
	}
	for res.Next() {
		var dummy, uJIDStr, oJIDStr string
		if err := res.Scan(&dummy, &uJIDStr, &oJIDStr); err != nil {
			return err
		}
		uJID, err := jid.NewWithString(uJIDStr, false)
		if err != nil {
			return err
		}
		oJID, err := jid.NewWithString(oJIDStr, false)
		if err != nil {
			return err
		}
		err = room.MapUserToOccupantJID(uJID, oJID)
		if err != nil {
			return err
		}
	}
	return nil
}

func fetchRoomInvites(ctx context.Context, tx *sql.Tx, room *mucmodel.Room,
	roomJID *jid.JID) error {
	resInv, err := sq.Select("room_jid", "user_jid").
		From("rooms_invites").
		Where(sq.Eq{"room_jid": roomJID.String()}).
		RunWith(tx).QueryContext(ctx)
	if err != nil {
		return err
	}
	for resInv.Next() {
		var dummy, uJIDStr string
		if err := resInv.Scan(&dummy, &uJIDStr); err != nil {
			return err
		}
		uJID, err := jid.NewWithString(uJIDStr, false)
		if err != nil {
			return err
		}
		err = room.InviteUser(uJID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *pgSQLRoom) DeleteRoom(ctx context.Context, roomJID *jid.JID) error {
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

func (r *pgSQLRoom) RoomExists(ctx context.Context, roomJID *jid.JID) (bool, error) {
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
