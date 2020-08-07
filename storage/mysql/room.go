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
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp/jid"
)

type mySQLRoom struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newRoom(db *sql.DB) *mySQLRoom {
	return &mySQLRoom{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (r *mySQLRoom) UpsertRoom(ctx context.Context, room *mucmodel.Room) error {
	// TODO this is pretty much a placeholder, needs to be filled out with all of the columns
	columns := []string{"name"}
	values := []interface{}{room.RoomJID.String()}

	q := sq.Insert("rooms").
		Columns(columns...).
		Values(values...)

	_, err := q.RunWith(r.db).ExecContext(ctx)
	return err
}

func (r *mySQLRoom) FetchRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	q := sq.Select("name").
		From("rooms").
		Where(sq.Eq{"name": roomJID.String()})

	var room mucmodel.Room

	err := q.RunWith(r.db).
		QueryRowContext(ctx).
		Scan(room.RoomJID)
	switch err {
	case nil:
		return &room, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *mySQLRoom) DeleteRoom(ctx context.Context, roomJID *jid.JID) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("rooms").Where(sq.Eq{"name": roomJID.String()}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *mySQLRoom) RoomExists(ctx context.Context, roomJID *jid.JID) (bool, error) {
	q := sq.Select("COUNT(*)").
		From("rooms").
		Where(sq.Eq{"roomName": roomJID.String()})

	var count int
	err := q.RunWith(r.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
