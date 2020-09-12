/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp/jid"
)

type pgSQLRoom struct {
	*pgSQLStorage
	pool *pool.BufferPool
}

func newRoom(db *sql.DB) *pgSQLRoom {
	return &pgSQLRoom{
		pgSQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

// UpsertRoom inserts a new room entity into storage, or updates it in case it's been previously inserted.
func (r *pgSQLRoom) UpsertRoom(ctx context.Context, room *mucmodel.Room) error {
	q := sq.Insert("rooms")

	q = q.Columns("name").
		Values(room.RoomJID.String())

	_, err := q.RunWith(r.db).ExecContext(ctx)
	return err
}

// FetchRoom retrieves from storage a room entity.
func (r *pgSQLRoom) FetchRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	q := sq.Select("name").
		From("rooms").
		Where(sq.Eq{"name": roomJID.String()})

	var room mucmodel.Room

	err := q.RunWith(r.db).QueryRowContext(ctx).Scan(room.RoomJID)
	switch err {
	case nil:
		return &room, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

// DeleteRoom deletes a room entity from storage.
func (r *pgSQLRoom) DeleteRoom(ctx context.Context, roomJID *jid.JID) error {
	return r.inTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("rooms").Where(sq.Eq{"name": roomJID.String()}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

// RoomExists returns whether or not a room exists within storage.
func (r *pgSQLRoom) RoomExists(ctx context.Context, roomJID *jid.JID) (bool, error) {
	var count int

	q := sq.Select("COUNT(*)").From("rooms").Where(sq.Eq{"name": roomJID.String()})
	err := q.RunWith(r.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}